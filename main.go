package main

import (
	"fmt"
	"os"

	"devbox-cli/cmd"
	"devbox-cli/internal/backup"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: devbox <command> [args]

Commands:
  version             Show the devbox CLI version

  setup               Configure/Change AWS credentials and region - stored in ~/.devbox/config.json
  clear-creds         Clear saved AWS credentials from ~/.devbox/config.json
  health              Check config, AWS credentials, region, and database

  create <name>       Create a new box
  ls                  List all boxes
  status <id|name>         Show details for a box
  rename <id|name> <new-name> Rename a box
  resize|upgrade <id|name> Resize a stopped box instance type or root disk
  stop <id|name>           Stop a running box
  start <id|name>          Start a stopped box
  restart|reboot <id|name> Reboot a running box
  delete <id|name>         Delete a box
  ssh [-i key] <id|name> [-- <ssh-option>...]  Open an SSH session to a box
                             -i  Path to SSH private key (default: ~/.ssh/id_ed25519)
                             --  Pass native ssh options/flags through, inserted before the
                                 target (e.g. -v, -A, -L 8080:localhost:8080); to run a
                                 one-off remote command instead, use "devbox exec"
  cp [-i key] <source> <dest> Copy a file to or from a box
                             Examples:
                               devbox cp ./main.go mybox:/home/ec2-user/app/
                               devbox cp mybox:/home/ec2-user/app/main.go ./
  sync [-i key] [--delete] <source> <dest> Sync files or directories to or from a box
                            --delete  Delete destination files missing from source
                             Examples:
                               devbox sync ./project mybox:/home/ec2-user/project
                               devbox sync mybox:/home/ec2-user/project ./project
  exec [-i key] [-s] [-t] <id|name> -- <command>
                             Run a one-off command on a running box
                             -s  Run as a shell snippet via sh -lc (for pipes, &&, cd);
                                 joins arguments and does not preserve per-arg boundaries
                             -t  Allocate a pseudo-TTY (for sudo / interactive commands)
  forward <id|name> <port> Forward a port from a box

  snapshot <id|name> <name>  Create a snapshot of a box
  snapshots                  List all your snapshots
  snapshots ls <amiId>       Show details for a specific snapshot
  snapshots delete <amiId>   Delete a snapshot
  create <name> [--from <snapshot_ami_id>]  Create a new box (optionally restore from snapshot)

  template                      List available templates
  template new <name> [command string] 			Create a new template with a command to run on startup
  template delete <id> 		 					Delete a template
  template rename <id> <new-name> 				Rename a template
  create --template <template> [<template>...] <name> Create a new box from one or more templates
  create --template <template> [<template>...] <name> --from <snapshot_ami_id> Create from templates and restore from a snapshot

  idle-stop <id|name> in <minutes> 			Stop the box after <minutes> minutes of inactivity
  idle-stop <id|name> show 					Show the idle stop for a box
  idle-stop <id|name> update <minutes> 		Update the idle stop for a box
  idle-stop <id|name> delete 				Delete the idle stop for a box
  `)
}

func main() {
	backup.MaybeDaily()
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "version":
		cmd.Version(args)
	case "setup":
		cmd.Setup(args)
	case "clear-creds":
		cmd.ClearCreds(args)
	case "health":
		cmd.Health(args)
	case "create":
		cmd.Create(args)
	case "ls":
		cmd.Ls(args)
	case "status":
		cmd.Status(args)
	case "rename":
		cmd.Rename(args)
	case "resize", "upgrade":
		cmd.Resize(args)
	case "stop":
		cmd.Stop(args)
	case "start":
		cmd.Start(args)
	case "restart", "reboot":
		cmd.Restart(args)
	case "delete":
		cmd.Delete(args)
	case "ssh":
		cmd.SSH(args)
	case "cp":
		cmd.CP(args)
	case "sync":
		cmd.Sync(args)
	case "exec":
		cmd.Exec(args)
	case "forward":
		cmd.Forward(args)
	case "snapshot":
		cmd.Snapshot(args)
	case "snapshots":
		cmd.Snapshots(args)
	case "template":
		cmd.Template(args)
	case "idle-stop":
		cmd.IdleRouter(args)
	default:
		fmt.Fprintf(os.Stderr, "devbox: unknown command %q\n\n", command)
		usage()
		os.Exit(1)
	}
}
