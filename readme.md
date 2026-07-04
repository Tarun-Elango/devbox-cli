# devbox-cli

Manage remote dev boxes from the CLI — provision, connect, sync, and destroy them with your own cloud account (BYOK).

- **Requirements:** AWS account, Linux or macOS
- **Usage:** run cli tool locally with an AWS access key and secret key (stored locally)

## What is a box?

A **box** is a personal dev machine on AWS — an EC2 instance running Amazon Linux that you provision, connect to, and tear down from your laptop.

## Why use devbox?

- **Dedicated dev machine on the cloud** — your own EC2 instance, separate from production and your daily driver
- **Smaller blast radius** — experiments, tooling, and dependencies stay off your main machine
- **Fast lifecycle** — create, use, and tear down boxes in minutes
- **Reproducible setups** — spin up consistent environments from templates
- **Build in commands for common tasks** — ssh, sync, idle-stop, git-sync
- **Secure** — AWS credentials and config stored locally on your machine

## Table of Contents
- [Download and Install (from GitHub release)](#download-and-install-from-github-release)
- [Setup](#setup)
- [Build using github repo](#build-using-github-repo)
- [Install (system-wide) using github repo](#install-system-wide-using-github-repo)
- [Common commands](#common-commands)
- [Local config](#notes-on-local-config-devbox)
- [AWS setup](#aws-setup)

## Download and Install (from GitHub release)

Every push to `main` publishes Linux and macOS binaries to the [latest release](https://github.com/Tarun-Elango/devbox-cli/releases/tag/latest). Run the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/devbox-cli/latest/scripts/install.sh | bash
```

Verify with the command `devbox ls`.

If that worked, you're done — skip the sections below. They're optional alternatives for pinning a version or installing system-wide.

#### Pin a specific version — To install a particular release instead of `latest`, set `RELEASE_TAG`:

```bash
RELEASE_TAG=v0.7.0 curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/devbox-cli/latest/scripts/install.sh | bash
```

#### Install system-wide — To install to `/usr/local/bin` (requires `sudo`, no shell config changes):

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/devbox-cli/latest/scripts/install.sh | sudo bash
```

## Setup

Run the interactive setup wizard to configure AWS credentials and local config, then create and connect to your first box:

```bash
devbox setup
devbox create mybox
devbox ssh mybox
```

Credentials are stored locally in your home directory.

## Build using github repo

```bash
go build -o devbox .
```

This produces a `devbox` binary.

To install it to your `$GOPATH/bin`:

```bash
go install .
```

To run the binary:
```bash
./devbox <command> [args]
```


## Install (system-wide) using github repo

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

---

## Common commands

Run `devbox help` to print usage, or see the table below.

### Config and health

| Command | Notes |
| --- | --- |
| `version` | Show the devbox CLI version |
| `update` | Check GitHub releases for a newer version and install it after confirmation |
| `setup` | Configure/change AWS credentials and region (stored in `~/.devbox/config.json`) |
| `clear-creds` | Clear saved AWS credentials from `~/.devbox/config.json` |
| `health` | Check config, AWS credentials, region, and database |

### Boxes

| Command | Notes |
| --- | --- |
| `create <name> [--from <amiId\|name>]` | Create a new box (optionally restore from a snapshot) |
| `create --template <template> [<template>...] <name> [--from <amiId\|name>]` | Create a box from one or more templates (optionally from snapshot) |
| `ls` | List all boxes |
| `status <id-or-name>` | Show details for a box |
| `rename <id-or-name> <new-name>` | Rename a box |
| `resize <id-or-name>` or `upgrade <id-or-name>` | Resize a stopped box instance type or root disk |
| `stop <id-or-name>` | Stop a running box |
| `start <id-or-name>` | Start a stopped box |
| `restart <id-or-name>` or `reboot <id-or-name>` | Reboot a running box |
| `delete <id-or-name>` | Delete a box |

### Connect and transfer

| Command | Notes |
| --- | --- |
| `ssh [-i key] <id-or-name> [-- <ssh-option>...]` | Open an SSH session to a box (`-i` path to private key; default `~/.ssh/id_ed25519`; `--` passes native ssh options before the target, e.g. `-v`, `-A`, `-L 8080:localhost:8080`; for one-off remote commands use `exec`) |
| `cp [-i key] <source> <dest>` | Copy a file to or from a box (e.g. `devbox cp ./main.go mybox:/home/ec2-user/app/`) |
| `sync [-i key] [--delete] <source> <dest>` | Sync files or directories to or from a box (`--delete` removes destination files missing from source) |
| `exec [-i key] [-s] [-t] <id-or-name> -- <command>` | Run a one-off command on a running box (`-s` run as a shell snippet via `sh -lc`; `-t` allocate a pseudo-TTY for sudo or interactive commands) |
| `forward <id-or-name> <port>` | Forward a port from a box |

### Snapshots

A snapshot is a saved disk image of a box; restore one with `create --from`.

| Command | Notes |
| --- | --- |
| `snapshot` | List all snapshots |
| `snapshot create <id-or-name> <name>` | Create a snapshot of a box |
| `snapshot ls <amiId-or-name>` | Show details for a snapshot |
| `snapshot delete <amiId-or-name>` | Delete a snapshot |

### Templates

Templates let you create boxes preloaded with libs, tools, and other setup.

| Command | Notes |
| --- | --- |
| `template` | List available templates |
| `template new <name> [command string]` | Create a template with optional startup command |
| `template delete <name>` | Delete a template |
| `template rename <name> <new-name>` | Rename a template |
| `template search <query>` | Search templates by name (returns partial matches) |

### Idle stop

| Command | Notes |
| --- | --- |
| `idle-stop set <id-or-name> <minutes>` | Stop the box after inactivity |
| `idle-stop show <id-or-name>` | Show idle-stop settings for a box |
| `idle-stop update <id-or-name> <minutes>` | Update idle-stop timeout |
| `idle-stop delete <id-or-name>` | Remove idle-stop from a box |

### Git sync

Use your local GitHub SSH key on a box (for `git push`, `git clone`, etc.) without copying it there: adds the key to `ssh-agent` and enables agent forwarding (`-A`) in the box's SSH config. Run again to undo both.

| Command | Notes |
| --- | --- |
| `git-sync <id-or-name>` | Toggle GitHub SSH agent forwarding for a box |

## Notes on local config (`~/.devbox`)

Credentials and tokens are stored in `~/.devbox/config.json` (mode 0600).
**Do not sync this folder** — not via dotfiles, iCloud, Dropbox, or Git.
Use a dedicated IAM user for AWS keys.

---

## AWS setup

Create a dedicated IAM user for devbox.

### 1. Create an IAM user

1. Open the IAM console → **Users** → **Create user**.
2. Enter a name (for example `devbox-cli`).
3. Choose **Attach policies directly**, search for `AmazonEC2FullAccess`, select it, and create the user.

### 2. Create access keys

1. Open the user → **Security credentials** → **Access keys** → **Create access key**.
2. Choose **Local code** and confirm.
3. Copy the **Access key ID** and **Secret access key** (the secret is shown only once).

### 3. Save credentials in devbox

```bash
devbox setup
```

Enter the access key, secret, and your preferred AWS region when prompted ( this will be stored in `~/.devbox/config.json` locally in your computer).