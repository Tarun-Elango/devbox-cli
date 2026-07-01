# devbox-cli

Manage remote dev boxes from the CLI — provision, connect, or destroy them.  
Support: linux, macos  
Infrastructure: AWS EC2  
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

Every push to `main` publishes Linux and macOS binaries to the [`latest` release](https://github.com/Tarun-Elango/devbox-cli/releases/tag/latest). The snippet below prints your OS and CPU architecture so you can pick the matching release asset and install it as `devbox`:

```bash
echo "Detected OS: $(uname -s), architecture: $(uname -m)"
# Linux x86_64  -> devbox-linux-amd64
# Linux aarch64 -> devbox-linux-arm64
# Linux arm64   -> devbox-linux-arm64
# macOS x86_64  -> devbox-darwin-amd64
# macOS arm64   -> devbox-darwin-arm64
curl -fsSL "https://github.com/Tarun-Elango/devbox-cli/releases/download/latest/devbox-<linux-or-darwin>-<amd64-or-arm64>" -o /tmp/devbox
chmod +x /tmp/devbox
sudo install -m 755 /tmp/devbox /usr/local/bin/devbox
```

Verify:

```bash
which devbox
devbox ls
```

**Without `sudo`** install to `~/.local/bin` instead (add it to your `PATH` if needed):

```bash
echo "Detected OS: $(uname -s), architecture: $(uname -m)"
# Linux x86_64  -> devbox-linux-amd64
# Linux aarch64 -> devbox-linux-arm64
# Linux arm64   -> devbox-linux-arm64
# macOS x86_64  -> devbox-darwin-amd64
# macOS arm64   -> devbox-darwin-arm64
mkdir -p ~/.local/bin
curl -fsSL "https://github.com/Tarun-Elango/devbox-cli/releases/download/latest/devbox-<linux-or-darwin>-<amd64-or-arm64>" -o ~/.local/bin/devbox
chmod +x ~/.local/bin/devbox
export PATH="$HOME/.local/bin:$PATH"
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
Use a dedicated low-privilege IAM user for AWS keys.

---

## AWS setup

Create a dedicated IAM user with only the EC2 permissions devbox needs: create boxes, view them, and delete them.

### 1. Create an IAM user

1. Open the IAM console → **Users** → **Create user**.
2. Enter a name (for example `devbox-cli`) and click **Next**.

### 2. Attach a minimal EC2 policy

1. Choose **Attach policies directly** → **Create policy** → **JSON**.
2. Paste the policy below, then save it (for example as `devbox-ec2-minimal`).
3. Back on the user-creation screen, refresh the policy list, select `devbox-ec2-minimal`, and finish creating the user.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DevboxEC2Create",
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:CreateTags",
        "ec2:CreateImage",
        "ec2:CreateSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:StartInstances"
      ],
      "Resource": "*"
    },
    {
      "Sid": "DevboxEC2View",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeImages",
        "ec2:DescribeVpcs",
        "ec2:DescribeSubnets",
        "ec2:DescribeSecurityGroups"
      ],
      "Resource": "*"
    },
    {
      "Sid": "DevboxEC2Delete",
      "Effect": "Allow",
      "Action": [
        "ec2:TerminateInstances",
        "ec2:StopInstances",
        "ec2:DeregisterImage",
        "ec2:DeleteSnapshot"
      ],
      "Resource": "*"
    }
  ]
}
```

This covers `create`, `ls`, `status`, `delete`, `start`, `stop`, and snapshot commands — nothing broader than required.

### 3. Create access keys

1. Open the user → **Security credentials** → **Access keys** → **Create access key**.
2. Choose **Command Line Interface (CLI)** and confirm.
3. Copy the **Access key ID** and **Secret access key** (the secret is shown only once).

### 4. Save credentials in devbox

```bash
devbox setup
```

Enter the access key, secret, and your preferred AWS region when prompted.