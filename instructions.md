# devbox-cli

## Build

```bash
cd src
go build -o devbox .
```

## Run

```bash
# Install so you can use it system-wide
go install .
```

## Test

```bash
# Run all tests
go test ./...
```

This produces a `devbox` binary in `src/`.

To install it to your `$GOPATH/bin`:

```bash
go install .
```

## Run

The CLI is a one-shot command — it performs an action and exits. It does **not** run a server or listen on any port (unless you use `forward`, which prints a forwarded URL and exits).

```
./devbox <command> [args]
```

**Common commands:**

| Command | Description |
|---|---|
| `login` | Authenticate (prompts for username/password) |
| `logout` | Clear saved credentials |
| `create [name]` | Create a new box |
| `ls` | List all boxes |
| `status <id>` | Show box details |
| `start <id>` / `stop <id>` | Start or stop a box |
| `delete <id>` | Delete a box |
| `ssh <id>` | Open an SSH session (`exec`s into ssh, replaces process) |
| `forward <id> <port>` | Request a port-forward and print the URL |
| `snapshot <id>` | Create a snapshot |
| `templates` | List available templates |

**Config** is stored in the default config directory (`~/.config/devbox/` on Linux/macOS). `login` saves the auth token there; all other commands read it automatically.

**Test mode** — prefix any command with `-test` to invoke the handler without making real API calls:

```bash
./devbox -test create mybox   # prints: [test] create: done
./devbox -test ssh abc123     # prints: [test] ssh: done
```
