# devbox-cli

## Build

From the project root:

```bash
go build -o devbox .
```

This produces a `devbox` binary in the current directory. Run it with `./devbox <command>`.

## Install (system-wide)

To install as `devbox` so you can run it from anywhere:

```bash
go build -o "$(go env GOPATH)/bin/devbox" .
```

Ensure `$GOPATH/bin` is on your PATH (default `~/go/bin`):

```bash
# telling shell to also look in the go path for the devbox binary
export GOPATH="${GOPATH:-$HOME/go}"
export PATH="$GOPATH/bin:$PATH"
```

Add those lines to `~/.bashrc` (or `~/.zshrc`) so they persist across sessions, then reload your shell:

```bash
# reload the shell
source ~/.bashrc
```

Verify:

```bash
which devbox
devbox ls
```

> **Note:** `go install .` also works but installs the binary as `devbox-cli` (from the module name), not `devbox`.

## Test

```bash
go test ./...
```

## Run

The CLI is a one-shot command — it performs an action and exits. It does **not** run a server or listen on any port (unless you use `forward`, which prints a forwarded URL and exits).

```
devbox <command> [args]
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
devbox -test create mybox   # prints: [test] create: done
devbox -test ssh abc123     # prints: [test] ssh: done
```
