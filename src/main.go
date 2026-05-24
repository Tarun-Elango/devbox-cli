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
	default:
		fmt.Fprintf(os.Stderr, "devbox: unknown command %q\n\n", command)
		usage()
		os.Exit(1)
	}
}
