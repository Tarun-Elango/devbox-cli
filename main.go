package main

import (
	"fmt"
	"os"

	"devbox-cli/cmd"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: devbox <command> [args]

Commands:
  login               Authenticate with the devbox server
  signup              Create a new account
  logout              Clear saved credentials
  create <name> [--from <snapshot_ami_id>]  Create a new box (optionally restore from snapshot)
  ls                  List all boxes
  status <id>         Show details for a box
  stop <id>           Stop a running box
  start <id>          Start a stopped box
  delete <id>         Delete a box
  ssh <id>            Open an SSH session to a box
  forward <id> <port> Forward a port from a box
  snapshot <id> [name]       Create a snapshot of a box
  snapshots                  List all your snapshots
  snapshots ls <boxId>       List snapshots for a specific box
  snapshots delete <amiId>   Delete a snapshot

  templates                  List available templates
  create --template <templateId> [<templateId>...] <name> Create a new box from one or more templates
  create --template <templateId> [<templateId>...] <name> --from <snapshot_ami_id> Create from templates and restore from a snapshot
  `)
}
/*
  reset
  email verification
  custom security group for each ssh 

  schedule (start|stop) <id> --at <RFC3339>              Start box once at time
  schedule (start|stop) <id> --cron "<expr>" [--tz TZ]   Recurring start (cron + optional IANA TZ)
  schedule (start|stop) <id> --daily HH:MM [--tz TZ]         Every day at HH:MM (optional sugar)
  schedule (start|stop) <id> --weekdays HH:MM [--tz TZ]      Mon–Fri at HH:MM
  schedule list                                   List schedules (id, box, action, next run, paused)
  schedule pause <scheduleId>                     Pause a schedule
  schedule resume <scheduleId>                    Resume a schedule
  schedule delete <scheduleId>                    Delete a schedule

  billing                      Show billing information
  billing hard-limit <amount>  Set a hard limit for the current month
*/

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
*/