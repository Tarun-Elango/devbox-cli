# devbox-cli

Manage remote dev boxes from the CLI — provision, connect, or destroy them.  
Support: linux, macos  
Requirements: Aws account, linux or macos
Usage: run locally with an AWS access key and secret key (stored locally)

## Table of Contents
- [Download and Install (from GitHub release)](#download-and-install-from-github-release)
- [Setup](#setup)
- [Build using github repo](#build-using-github-repo)
- [Install (system-wide) using github repo](#install-system-wide-using-github-repo)
- [Common commands](#common-commands)
- [Local config](#notes-on-local-config-devbox)
- [AWS setup](#aws-setup)

## Download and Install (from GitHub release)

Every push to `main` publishes Linux and macOS binaries to the [`latest` release](https://github.com/Tarun-Elango/devbox-cli/releases/tag/latest).

```bash
curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/devbox-cli/main/scripts/install.sh | bash
```

This detects your OS and CPU, downloads the matching binary, installs it to `~/.local/bin`, and adds that directory to your shell config if needed. Restart your shell, then verify:

```bash
devbox ls
```

To install system-wide (no shell config changes — `/usr/local/bin` is usually already on PATH):

Same script as above, but installs to `/usr/local/bin` for all users on the machine; requires `sudo`, and skips editing your shell config.

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/devbox-cli/main/scripts/install.sh | sudo bash
```

## Setup

Run the interactive setup wizard to configure AWS credentials and local config:

```bash
devbox setup
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

Run `devbox` with no arguments to print usage, or see the table below.

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
| `create <name>` | Create a new box |
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
| `ssh [-i key] <id-or-name> [-- <ssh-option>...]` | Open an SSH session to a box (`-i` path to private key; default `~/.ssh/id_ed25519`) |
| `cp [-i key] <source> <dest>` | Copy a file to or from a box (e.g. `devbox cp ./main.go mybox:/home/ec2-user/app/`) |
| `sync [-i key] [--delete] <source> <dest>` | Sync files or directories to or from a box (`--delete` removes destination files missing from source) |
| `exec [-i key] [-s] [-t] <id-or-name> -- <command>` | Run a one-off command on a running box (`-s` run through `sh`; `-t` allocate a TTY) |
| `forward <id-or-name> <port>` | Forward a port from a box |

### Snapshots

| Command | Notes |
| --- | --- |
| `snapshot <id-or-name> <name>` | Create a snapshot of a box |
| `snapshots` | List all your snapshots |
| `snapshots ls <amiId>` | Show details for a specific snapshot |
| `snapshots delete <amiId>` | Delete a snapshot |
| `create <name> [--from <snapshot_ami_id>]` | Create a new box (optionally restore from a snapshot) |

### Templates

| Command | Notes |
| --- | --- |
| `template` | List available templates |
| `template new <name> [command string]` | Create a new template with a command to run on startup |
| `template delete <id>` | Delete a template |
| `template rename <id> <new-name>` | Rename a template |
| `create --template <template> [<template>...] <name>` | Create a new box from one or more templates |
| `create --template <template> [<template>...] <name> --from <snapshot_ami_id>` | Create from templates and restore from a snapshot |

### Idle stop

| Command | Notes |
| --- | --- |
| `idle-stop <id-or-name> in <minutes>` | Stop the box after `<minutes>` minutes of inactivity |
| `idle-stop <id-or-name> show` | Show the idle stop for a box |
| `idle-stop <id-or-name> update <minutes>` | Update the idle stop for a box |
| `idle-stop <id-or-name> delete` | Delete the idle stop for a box |

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