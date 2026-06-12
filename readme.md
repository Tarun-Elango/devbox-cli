# devbox-cli

Manage remote dev boxes from the CLI — provision, connect, or destroy them. 

## Build

```bash
go build -o devbox .
```

This produces a `devbox` binary.

To install it to your `$GOPATH/bin`:

```bash
go install .
```

## Run

The CLI is a one-shot command — it performs an action and exits.
```
./devbox <command> [args]
```

**Common commands:**

| Command                                    | Notes (what it does)                                    |
| ------------------------------------------ | ------------------------------------------------------- |
| `setup`                                    | Configure/change AWS credentials and region (stored in `~/.devbox/config.json`) |
| `create <name>`                            | Create a new box                                        |
| `ls`                                       | List all boxes                                          |
| `status <id>`                              | Show details for a box                                  |
| `stop <id>`                                | Stop a running box                                      |
| `start <id>`                               | Start a stopped box                                     |
| `delete <id>`                              | Delete a box                                            |
| `ssh <id>`                                 | Open an SSH session to a box                            |
| `forward <id> <port>`                      | Forward a port from a box                               |
| `snapshot <id> [name]`                     | Create a snapshot of a box                              |
| `snapshots`                                | List all your snapshots                                 |
| `snapshots ls <amiId>`                     | Show details for a specific snapshot                    |
| `snapshots delete <amiId>`                 | Delete a snapshot                                       |
| `create <name> [--from <snapshot_ami_id>]` | Create a new box (optionally restore from a snapshot)   |
| `templates`                                | List available templates                                |
| `template new <name> [command string]`     | Create a new template with a command to run on startup  |
| `template delete <id>`                     | Delete a template                                       |
| `create --template <template> [<template>...] <name>` | Create a new box from one or more templates |
| `create --template <template> [<template>...] <name> --from <snapshot_ami_id>` | Create from templates and restore from a snapshot |


**Config** is stored in the default config directory (`~/.config/devbox/` on Linux/macOS). `login` saves the auth token there; all other commands read it automatically.

**Test mode** — prefix any command with `-test` to invoke the handler without making real API calls:

```bash
./devbox -test create mybox   # prints: [test] create: done
./devbox -test ssh abc123     # prints: [test] ssh: done
```
