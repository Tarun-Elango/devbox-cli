# devbox CLI — Plan

## Overview
A CLI tool that communicates with a remote devbox server. Auth is token-based (stored locally). All commands map to REST endpoints, with WebSocket polling for async operations.

---

## Tech Stack
- **Language**: Go — compiles to a single static binary, zero install for users
- **CLI framework**: none — stdlib `os.Args` + `flag` for argument parsing
- **HTTP client**: stdlib `net/http`
- **WebSocket**: hand-rolled minimal WS client using stdlib `net/http` hijack + `bufio` (no external deps)
- **Config/token storage**: `~/.devbox/config.json`
- **Zero external dependencies** — pure stdlib only
- **Distribution**: pre-built binaries on GitHub Releases, install via:
  ```sh
  curl -fsSL https://github.com/yourorg/devbox-cli/releases/latest/download/devbox_$(uname -s)_$(uname -m) -o /usr/local/bin/devbox && chmod +x /usr/local/bin/devbox
  ```

---

## Project Structure
```
devbox-cli/
├── main.go                    # Entry point — dispatches on os.Args[1]
├── cmd/
│   ├── auth.go                # login, logout
│   ├── boxes.go               # create, ls, status, stop, start, delete
│   ├── ssh.go                 # ssh
│   ├── forward.go             # forward
│   ├── snapshot.go            # snapshot
│   └── templates.go           # templates
├── internal/
│   ├── api/
│   │   └── client.go          # net/http wrapper, injects auth header
│   ├── config/
│   │   └── config.go          # read/write ~/.devbox/config.json
│   └── ws/
│       └── ws.go              # hand-rolled minimal WebSocket client (stdlib only)
└── go.mod                     # no external deps, module declaration only
```

### Dispatch pattern (`main.go`)
```go
switch os.Args[1] {
case "login":    cmd.Login(os.Args[2:])
case "logout":   cmd.Logout()
case "create":   cmd.Create(os.Args[2:])
case "ls":       cmd.Ls()
// ...
default:         usage(); os.Exit(1)
}
```

---

## Commands

| Command | Method | Endpoint | Notes |
|---|---|---|---|
| `devbox login` | POST | `/v1/auth/login` | Prompts user/pass, saves token |
| `devbox logout` | POST | `/v1/auth/logout` | Clears saved token |
| `devbox create [name]` | POST | `/v1/boxes` | Polls status via WebSocket |
| `devbox ls` | GET | `/v1/boxes` | Lists all boxes |
| `devbox status <id>` | GET | `/v1/boxes/:id` | Shows box details |
| `devbox ssh <id>` | GET | `/v1/boxes/:id` | Fetches IP, execs `ssh` locally |
| `devbox stop <id>` | POST | `/v1/boxes/:id/stop` | |
| `devbox start <id>` | POST | `/v1/boxes/:id/start` | |
| `devbox delete <id>` | DELETE | `/v1/boxes/:id` | |
| `devbox forward <id> <port>` | POST | `/v1/boxes/:id/ports` | Prints forwarded URL |
| `devbox snapshot <id>` | POST | `/v1/boxes/:id/snapshots` | |
| `devbox templates` | GET | `/v1/boxes/templates` | Lists available templates |

---

## SSH Key Injection Flow

No AWS credentials are required on the client. The server owns the EC2 lifecycle; the CLI only needs the user's public key at create-time and the private key at connect-time.

### `devbox create <name>`
1. CLI reads `~/.ssh/id_ed25519.pub` (falls back to `~/.ssh/id_rsa.pub`) and includes it as `public_key` in the POST `/v1/boxes` body.
2. Server creates the EC2 with a fixed security group (port 22 open to `0.0.0.0/0`).
3. Server injects the public key into `~/.ssh/authorized_keys` and disables `PasswordAuthentication` via EC2 user-data.
4. CLI streams provisioning progress over WebSocket until the box reaches `running`.

### `devbox ssh <id>`
1. CLI fetches the box IP via GET `/v1/boxes/:id`.
2. CLI execs `ssh -i ~/.ssh/id_ed25519 -p 22 <user>@<ip>`, replacing the current process.
3. The `-i` key path defaults to `id_ed25519` → `id_rsa` (whichever exists) and can be overridden with the `-i` flag.

```
devbox create mybox   →  POST /v1/boxes  { "name": "mybox", "public_key": "<pubkey>" }
devbox ssh mybox      →  exec ssh -i ~/.ssh/id_ed25519 root@<ip>
```

---

## Config
Token stored at `~/.devbox/config.json`:
```json
{ "token": "<jwt>", "serverUrl": "https://api.devbox.io" }
```
`client.go` reads this and injects `Authorization: Bearer <token>` on every request.

---

## Placeholders
- All API calls will return **mock/stub responses** until the server is built.
- WebSocket polling in `ws.go` will be a no-op stub that immediately resolves.
- `devbox ssh` will stub the IP and skip the actual `ssh` exec.

---

## Implementation Order
1. `go.mod` init (no deps) + `os.Args` dispatch in `main.go`
2. `config.go` + `client.go` (foundation for all commands)
3. `auth.go` — login / logout
4. `boxes.go` — ls, status, create, stop, start, delete
5. `ssh.go`, `forward.go`, `snapshot.go`, `templates.go`
6. `ws.go` stub for create polling
7. GitHub Actions workflow → cross-compile for macOS/Linux/Windows + attach to release
