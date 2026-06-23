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

## Release a new version

Releases are **not** created on every merge to `main`. Publish when you're ready:

1. **Bump the version** in `internal/version/VERSION` (e.g. `0.1.0` → `0.1.1`).

2. **Commit and merge** that change to `main` (on its own or as part of your release PR).

3. **Run the release workflow** on GitHub:

   - Go to **Actions** → **Build and Publish Release**
   - Click **Run workflow**
   - Branch: `main` (default)
   - Click **Run workflow**

   The workflow reads `internal/version/VERSION`, creates tag `v0.1.1` on that commit, builds Linux/macOS binaries (amd64 and arm64), and publishes the GitHub release.

4. **Verify** the release under **Releases**, or after installing that build:

   ```bash
   devbox version
   ```

> **Note:** Merging to `main` without running the workflow does nothing to existing releases. Each release needs a new version in `internal/version/VERSION` — if tag `v0.1.1` already exists, the workflow fails until you bump the file.
