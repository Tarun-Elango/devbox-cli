# devbox-cli

Manage remote dev boxes from the CLI — provision, connect, or destroy them. 

## Table of Contents
- [Download and Install (from GitHub release)](#download-and-install-from-github-release)
- [Build using github repo](#build-using-github-repo)
- [Install (system-wide) using github repo](#install-system-wide-using-github-repo)
- [Common commands](#common-commands)
- [Local config](#notes-on-local-config-devbox)
- [AWS setup](#aws-setup)

## Download and Install (from GitHub release)

Every push to `main` publishes Linux and macOS binaries to the [`latest` release](https://github.com/Tarun-Elango/devboxssh-cli/releases/tag/latest). The snippet below prints your OS and CPU architecture so you can pick the matching release asset and install it as `devbox`:

```bash
echo "Detected OS: $(uname -s), architecture: $(uname -m)"
# Linux x86_64  -> devbox-linux-amd64
# Linux aarch64 -> devbox-linux-arm64
# Linux arm64   -> devbox-linux-arm64
# macOS x86_64  -> devbox-darwin-amd64
# macOS arm64   -> devbox-darwin-arm64
curl -fsSL "https://github.com/Tarun-Elango/devboxssh-cli/releases/download/latest/devbox-<linux-or-darwin>-<amd64-or-arm64>" -o /tmp/devbox
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
curl -fsSL "https://github.com/Tarun-Elango/devboxssh-cli/releases/download/latest/devbox-<linux-or-darwin>-<amd64-or-arm64>" -o ~/.local/bin/devbox
chmod +x ~/.local/bin/devbox
export PATH="$HOME/.local/bin:$PATH"
```

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

## Common commands:

usage method: devbox command

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
| `idle-stop <id> in <minutes>` | Stop the box after <minutes> minutes of inactivity |
| `idle-stop <id> show` | Show the idle stop for a box |
| `idle-stop <id> update <minutes>` | Update the idle stop for a box |
| `idle-stop <id> delete` | Delete the idle stop for a box |

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