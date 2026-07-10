# outpost CLI

Manage remote dev boxes from the CLI — provision, connect, sync, and destroy them using your own cloud account (BYOK - AWS only for now). CLI supports linux and macos.

## What is a box?

A **box** is a personal dev machine on AWS — an EC2 instance running Amazon Linux that you provision, connect to, and tear down from your local machine.

## Why was this created?
The idea of keeping your dev environment away from your machine, keeps the blast radius small when installing any new packages or dependencies. Having a seperate environment for AI agents to do their thing, without having to worry about our computer. All while staying in the terminal, and keeping credentials secure locally.

The tool has since been extended with commands for box management, snapshots (save a copy of your box so you can restore it later), templates (spin up a box with pre-installed software), ssh, sync, idle-stop to save on costs, git-sync to use your local git credentials on the box, budget tracking, and more.

see the docs for more details: https://outpost.tarunelango.com

## Table of Contents
- [Download and Install (from GitHub release)](#download-and-install-from-github-release)
- [Common commands](#common-commands)
- [Setup](#setup)

## Download and Install (from GitHub release)

Every push to `main` publishes Linux and macOS binaries to the [latest release](https://github.com/Tarun-Elango/outpost/releases/tag/latest). Run the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/outpost/latest/scripts/install.sh | bash
```

Verify with the command `outpost ls`.

If that worked, you're done — skip the sections below. They're optional alternatives for pinning a version or installing system-wide.

Note: if you want to pin a specific version, or install it for every user on this machine, see here https://outpost.tarunelango.com/docs/install

If you want to clone the repo and build it yourself, see here https://outpost.tarunelango.com/docs/install#build-from-source

## Common commands

Run `outpost help` to print usage, or see the table below.

### Config and health

| Command | Notes |
| --- | --- |
| `version` | Show the outpost CLI version |
| `update` | Check GitHub releases for a newer version and install it after confirmation |
| `setup` | Configure/change AWS credentials and region (stored in `~/.outpost/config.json`) |
| `clear-creds` | Clear saved AWS credentials from `~/.outpost/config.json` |
| `health` | Check config, AWS credentials, region, and database |

### Boxes

| Command | Notes |
| --- | --- |
| `create <name> [--template <templateName>...] [--from <amiId\|name>]` | Create a new box, optionally from one or more templates, optionally restoring from a snapshot |
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
| `cp [-i key] <source> <dest>` | Copy a file to or from a box (e.g. `outpost cp ./main.go mybox:/home/ec2-user/app/`) |
| `sync [-i key] [--delete] <source> <dest>` | Incremental directory sync via rsync over SSH (same path syntax as `cp`: one local path, one `box:/path`). Only **dest** is modified — copies new/changed files from source; source is never changed. `--delete` also removes files on dest that are not in source |
| `exec [-i key] [-s] [-t] <id-or-name> -- <command>` | Run a one-off command on a running box (`-s` run as a shell snippet via `sh -lc`; `-t` allocate a pseudo-TTY for sudo or interactive commands) |
| `forward <id-or-name> <port>` | Forward a port from a box |

### Snapshots

A snapshot is a saved disk image of a box; restore one with `create --from`.

| Command | Notes |
| --- | --- |
| `snapshot [ls] [<amiId-or-name>]` | List all snapshots, or show details for one |
| `snapshot create <id-or-name> <name>` | Create a snapshot of a box |
| `snapshot delete <amiId-or-name>` | Delete a snapshot |

### Templates

Templates let you create boxes preloaded with libs, tools, and other setup.

| Command | Notes |
| --- | --- |
| `template [ls]` | List available templates |
| `template new <templateName> [command string]` | Create a template with optional startup command |
| `template delete <templateName>` | Delete a template |
| `template rename <templateName> <new-templateName>` | Rename a template |
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

### Budgets

List and manage AWS account cost budgets. Results are cached under `~/.outpost/` for 12 hours. Requires the `AWSBudgetsActionsWithAWSResourceControlAccess` IAM policy.

| Command | Notes |
| --- | --- |
| `budget [ls] [--refresh]` | List AWS account budgets (name, period, limit, spend, forecast, % of budget). `--refresh` bypasses the local cache |
| `budget create <name> <limit> <email>` | Create a monthly cost budget for all AWS services. Alerts at 85% actual, 100% actual, and 100% forecasted spend |
| `budget update <name>` | Interactively update name, limit, or alert email (Enter keeps each current value) |
| `budget delete <name>` | Delete a budget by exact name (quote names with spaces) |

## Setup

***Note***: the aws access key and secret are stored in the `~/.outpost/` file, locally on your machine. No cloud storage or syncing is done.

Go to your AWS console 

1. **create an IAM user**
Go to the IAM console → **Users** → **Create user**. Name it (e.g. `outpost`), choose **Attach policies directly**, search for `AmazonEC2FullAccess` and `AWSBudgetsActionsWithAWSResourceControlAccess`, select it, and create the user.

2. **Access keys** 
Open the user → **Security credentials** → **Access keys** → **Create access key**. Select **Local code**, confirm, then copy the **Access key ID** and **Secret access key** (the secret is shown only once).

3. **Configure outpost**
Run the steps below to create your first box:

```bash
outpost setup    # enter access key, secret, region → ~/.outpost/config.json
outpost create mybox  # create a simple box
outpost ssh mybox  # ssh into the box
```

for detailed instructions, see here https://outpost.tarunelango.com/docs/setup