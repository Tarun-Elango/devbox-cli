# devbox Server — Spring Boot Implementation Plan

## Overview
REST + WebSocket backend that the devbox CLI talks to. Manages user auth, box lifecycle on EC2, SSH key injection, port forwarding, and snapshots.

---

## Tech Stack
- **Framework**: Spring Boot 3.x
- **Language**: Java 21 (or Kotlin)
- **Auth**: Spring Security + JWT (JJWT or nimbus-jose)
- **DB**: PostgreSQL — users, boxes, snapshots metadata
- **ORM**: Spring Data JPA
- **AWS**: AWS SDK v2 — EC2, EC2 Instance Connect (key injection)
- **WebSocket**: Spring WebSocket (`@EnableWebSocket`)
- **Build**: Maven or Gradle

---

## Project Structure
```

only a sample structure to illustrate organization; actual package names may vary

devbox-server/
├── src/main/java/io/devbox/
│   ├── DevboxApplication.java
│   ├── config/
│   │   ├── SecurityConfig.java        # JWT filter, CORS, stateless session
│   │   └── WebSocketConfig.java       # WS endpoint registration
│   ├── auth/
│   │   ├── AuthController.java        # POST /v1/auth/login, /logout
│   │   ├── AuthService.java
│   │   ├── JwtUtil.java
│   │   └── UserRepository.java
│   ├── boxes/
│   │   ├── BoxController.java         # All /v1/boxes/** REST endpoints
│   │   ├── BoxService.java            # Orchestrates EC2 + DB
│   │   ├── BoxRepository.java
│   │   └── Box.java                   # JPA entity
│   ├── ec2/
│   │   ├── Ec2Service.java            # AWS SDK: runInstances, stop, start, terminate
│   │   └── UserDataBuilder.java       # Builds cloud-init script for key injection
│   ├── ws/
│   │   ├── BoxStatusHandler.java      # WS handler — streams provisioning events
│   │   └── ProvisioningTask.java      # Async polling EC2 until running
│   ├── forward/
│   │   ├── ForwardController.java     # POST /v1/boxes/:id/ports
│   │   └── ForwardService.java        # Records port mapping, returns URL
│   ├── snapshot/
│   │   ├── SnapshotController.java    # POST /v1/boxes/:id/snapshots
│   │   └── SnapshotService.java       # EC2 CreateImage API
│   └── templates/
│       └── TemplateController.java    # GET /v1/boxes/templates (static list or DB)
└── src/main/resources/
    ├── application.yml
    └── db/migration/                  # Flyway migrations
```

---

## API Endpoints

| Method | Path | Description |
|---|---|---|
| POST | `/v1/auth/login` | Validates credentials, returns JWT |
| POST | `/v1/auth/logout` | Invalidates token (blocklist or stateless no-op) |
| GET | `/v1/boxes` | List boxes for authenticated user |
| POST | `/v1/boxes` | Create box — accepts `{ name, public_key }` |
| GET | `/v1/boxes/:id` | Get box details (id, name, ip, status) |
| POST | `/v1/boxes/:id/start` | Start stopped EC2 instance |
| POST | `/v1/boxes/:id/stop` | Stop running EC2 instance |
| DELETE | `/v1/boxes/:id` | Terminate EC2, delete DB record |
| POST | `/v1/boxes/:id/ports` | Register port forward, return URL |
| POST | `/v1/boxes/:id/snapshots` | Create EC2 AMI snapshot |
| GET | `/v1/boxes/templates` | List available AMI templates |
| WS | `/v1/boxes/:id/status` | Stream provisioning events until `running` |

---

## Data Model

### `users`
| Column | Type |
|---|---|
| id | UUID PK |
| username | VARCHAR UNIQUE |
| password_hash | VARCHAR |
| created_at | TIMESTAMP |

### `boxes`
| Column | Type |
|---|---|
| id | UUID PK |
| user_id | UUID FK → users |
| name | VARCHAR |
| ec2_instance_id | VARCHAR |
| ip_address | VARCHAR |
| status | ENUM(pending, running, stopped, terminated) |
| created_at | TIMESTAMP |

### `snapshots`
| Column | Type |
|---|---|
| id | UUID PK |
| box_id | UUID FK → boxes |
| ami_id | VARCHAR |
| created_at | TIMESTAMP |

---

## Key Implementation Details

### Auth Flow
1. `POST /v1/auth/login` — verify bcrypt password, sign JWT (HS256, 24h expiry), return `{ token }`.
2. All protected routes — `JwtAuthFilter` extracts Bearer token, validates, loads `UserDetails`.
3. Logout — client discards token; optionally maintain a Redis blocklist for immediate revocation.

### Box Creation Flow
1. Controller receives `{ name, public_key }`.
2. `BoxService.create()`:
   - Persists box record with status `pending`.
   - Calls `Ec2Service.runInstance(publicKey)` which builds a `user-data` script:
     ```bash
     #!/bin/bash
     mkdir -p /root/.ssh
     echo "<public_key>" >> /root/.ssh/authorized_keys
     chmod 600 /root/.ssh/authorized_keys
     sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
     systemctl restart sshd
     ```
   - Returns `instanceId` immediately; starts async `ProvisioningTask`.
3. `ProvisioningTask` polls `ec2.describeInstances` every 5s until state = `running`, then updates box IP + status in DB and pushes event over WebSocket.

### WebSocket Provisioning Stream
- Client connects to `ws://<host>/v1/boxes/:id/status` after POST create.
- Server sends JSON frames: `{ "status": "pending" }` → `{ "status": "running", "ip": "1.2.3.4" }`.
- Closes connection once `running` or `error`.

### EC2 Config (application.yml)
```yaml
aws:
  region: us-east-1
  ami-id: ami-0abcdef1234567890   # base Ubuntu AMI
  instance-type: t3.micro
  security-group-id: sg-xxxxxxxx  # port 22 open to 0.0.0.0/0
  key-name: ""                    # no managed keypair — public key injected via user-data

devbox:
  jwt-secret: <secret>
  jwt-expiry-hours: 24
```

---

## Implementation Order
1. Project scaffold — Spring Initializr (Web, Security, Data JPA, WebSocket, Flyway, PostgreSQL)
2. `users` migration + `AuthController` + JWT filter — login/logout working end-to-end
3. `boxes` migration + `BoxController` GET/POST/DELETE stubs (no EC2 yet, mock status)
4. `Ec2Service` — real `runInstances`, `stopInstances`, `startInstances`, `terminateInstances`
5. `UserDataBuilder` + SSH key injection wired into create flow
6. `ProvisioningTask` async polling + `BoxStatusHandler` WebSocket stream
7. `ForwardController`, `SnapshotController`, `TemplateController`
8. Integration tests with LocalStack for EC2 mocks
9. Dockerfile + docker-compose (app + postgres)

---

## Security Notes
- Store `jwt-secret` in env var / AWS Secrets Manager — never in source.
- Scope all box queries to `user_id` from JWT to prevent IDOR.
- Validate `public_key` format server-side before injecting into user-data.
- Use parameterized queries only (JPA handles this).
- Rate-limit `/v1/auth/login` to prevent brute force.
