package main

import (
	"fmt"
	"os"

	"devbox-cli/cmd"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: devbox <command> [args]

Commands:
  setup               Configure/Change AWS credentials and region - stored in ~/.devbox/config.json

  create <name>       Create a new box
  ls                  List all boxes
  status <id>         Show details for a box
  stop <id>           Stop a running box
  start <id>          Start a stopped box
  delete <id>         Delete a box
  ssh <id>            Open an SSH session to a box
  forward <id> <port> Forward a port from a box

  snapshot <id> [name]       Create a snapshot of a box
  snapshots                  List all your snapshots
  snapshots ls <amiId>       Show details for a specific snapshot
  snapshots delete <amiId>   Delete a snapshot
  create <name> [--from <snapshot_ami_id>]  Create a new box (optionally restore from snapshot)

  templates                  List available templates
  template new <name> [command string] Create a new template with a command to run on startup
  template delete <id> 		 Delete a template
  create --template <template> [<template>...] <name> Create a new box from one or more templates
  create --template <template> [<template>...] <name> --from <snapshot_ami_id> Create from templates and restore from a snapshot
  `)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	if command == "-test" {
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		cmd.TestMode = true
		command = os.Args[2]
		args = os.Args[3:]
	}

	switch command {
	case "mode":
		cmd.Mode(args)
	case "setup":
		cmd.Setup(args)
	case "login":
		cmd.Login(args)
	case "signup":
		cmd.Signup(args)
	case "logout":
		cmd.Logout()
	case "create":
		cmd.Create(args)
	case "ls":
		cmd.Ls()
	case "status":
		cmd.Status(args)
	case "stop":
		cmd.Stop(args)
	case "start":
		cmd.Start(args)
	case "delete":
		cmd.Delete(args)
	case "ssh":
		cmd.SSH(args)
	case "forward":
		cmd.Forward(args)
	case "snapshot":
		cmd.Snapshot(args)
	case "snapshots":
		cmd.Snapshots(args)
	case "templates":
		cmd.Templates(args)
	case "template":
		cmd.Template(args)
	default:
		fmt.Fprintf(os.Stderr, "devbox: unknown command %q\n\n", command)
		usage()
		os.Exit(1)
	}
}

/*
# One-shot: stop tonight at 6pm UTC
devbox schedule stop i-0abc123 --at 2026-05-27T18:00:00Z

# One-shot: start tomorrow 9am in local TZ
devbox schedule start i-0abc123 --at 2026-05-28T09:00:00-05:00

# Recurring: stop weekdays at 6pm, start weekdays at 9am (two schedules)
devbox schedule stop  i-0abc123 --cron "0 18 * * MON-FRI" --tz America/New_York
devbox schedule start i-0abc123 --cron "0 9 * * MON-FRI"  --tz America/New_York


┌──────── minute (0–59)
│ ┌────── hour (0–23)
│ │ ┌──── day of month (*)
│ │ │ ┌── month (*)
│ │ │ │ ┌ day of week (MON–SUN or 0–6)
│ │ │ │ │
0 18 * * *        → every day 18:00
0 9  * * MON-FRI  → Mon–Fri 09:00


  mode                Set the mode to cloud or local (default is local)
*/
