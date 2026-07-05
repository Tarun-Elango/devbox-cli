package main

import (
	"fmt"
	"os"

	"devbox-cli/cmd"
	"devbox-cli/internal/backup"
)

const helpTopics = "create, box, ssh, snapshot, template, idle-stop, git-sync"

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: devbox <command> [args]

Commands:
  help, -h, --help    Show this help message
  help <topic>        Show help for a topic (%s)

  version             Show the devbox CLI version
  update              Check for a newer release and install it (asks for confirmation)

  setup               Configure AWS credentials and region (stored in ~/.devbox/config.json)
  clear-creds         Clear saved AWS credentials from ~/.devbox/config.json
  health              Check config, AWS credentials, region, and database

  create <name> [--template <templateName>...] [--from <amiId|name>]
                      Create a new box (optionally from one or more templates,
                      and optionally restore from a snapshot)

  ls                  List all boxes
  status <id|name>    Show details for a box
  rename <id|name> <new-name>
                      Rename a box
  resize|upgrade <id|name>
                      Resize a stopped box instance type or root disk
  stop <id|name>      Stop a running box
  start <id|name>     Start a stopped box
  restart|reboot <id|name>
                      Reboot a running box
  delete <id|name>    Delete a box

  ssh [-i key] <id|name> [-- <ssh-option>...]
                      Open an SSH session to a box
                        -i  Path to SSH private key (default: ~/.ssh/id_ed25519)
                        --  Pass native ssh options before the target (e.g. -v, -A,
                            -L 8080:localhost:8080); for one-off remote commands use exec
  cp [-i key] <source> <dest>
                      Copy a file to or from a box
                        devbox cp ./main.go mybox:/home/ec2-user/app/
                        devbox cp mybox:/home/ec2-user/app/main.go ./
  sync [-i key] [--delete] <source> <dest>
                      Sync directories via rsync over SSH (one local path, one box:/path)
                        Only dest is modified; source is read-only.
                        --delete also removes files on dest that are not in source
                        devbox sync ./project mybox:/home/ec2-user/project
                        devbox sync mybox:/home/ec2-user/project ./project
  exec [-i key] [-s] [-t] <id|name> -- <command>
                      Run a one-off command on a running box
                        -s  Run as a shell snippet via sh -lc (pipes, &&, cd)
                        -t  Allocate a pseudo-TTY (for sudo / interactive commands)
  forward <id|name> <port>
                      Forward a port from a box

  snapshot                              List all snapshots
  snapshot create <id|name> <name>      Create a snapshot of a box
  snapshot ls <amiId|name>              Show details for a snapshot
  snapshot delete <amiId|name>          Delete a snapshot

  template                              List available templates
  template new <templateName> [command string]  Create a template with optional startup command
  template delete <templateName>                Delete a template
  template rename <templateName> <new-templateName>     Rename a template
  template search <query>               Search templates by name ( returns partial matches )

  idle-stop set <id|name> <minutes>     Stop the box after inactivity
  idle-stop show <id|name>              Show idle-stop settings for a box
  idle-stop update <id|name> <minutes>  Update idle-stop timeout
  idle-stop delete <id|name>            Remove idle-stop from a box

  git-sync <id|name>  Toggle GitHub SSH access for a box: adds the local key
                      to ssh-agent and enables agent forwarding (-A) in the
                      box's SSH config; run again to undo both.
`, helpTopics)
}

func helpCreate() {
	fmt.Fprintf(os.Stderr, `Usage: devbox create <name> [--template <templateName>...] [--from <amiId|name>]

  create <name>
                      Create a new box
  create <name> --template <templateName> [<templateName>...]
                      Create a box from one or more templates
  create <name> [--from <amiId|name>]
                      Restore a box from a previously saved snapshot
  create <name> --template <templateName> [<templateName>...] --from <amiId|name>
                      Create a box from templates, restoring from a snapshot
`)
}

func helpBox() {
	fmt.Fprintf(os.Stderr, `Usage: devbox <box-command> <id|name> [args]

  ls                  List all boxes
  status <id|name>    Show details for a box
  rename <id|name> <new-name>
                      Rename a box
  resize|upgrade <id|name>
                      Resize a stopped box instance type or root disk
  stop <id|name>      Stop a running box
  start <id|name>     Start a stopped box
  restart|reboot <id|name>
                      Reboot a running box
  delete <id|name>    Delete a box
`)
}

func helpSSH() {
	fmt.Fprintf(os.Stderr, `Usage: devbox <ssh-command> [args]

  ssh [-i key] <id|name> [-- <ssh-option>...]
                      Open an SSH session to a box
                        -i  Path to SSH private key (default: ~/.ssh/id_ed25519)
                        --  Pass native ssh options before the target (e.g. -v, -A,
                            -L 8080:localhost:8080); for one-off remote commands use exec
  cp [-i key] <source> <dest>
                      Copy a file to or from a box
                        devbox cp ./main.go mybox:/home/ec2-user/app/
                        devbox cp mybox:/home/ec2-user/app/main.go ./
  sync [-i key] [--delete] <source> <dest>
                      Sync directories via rsync over SSH (one local path, one box:/path)
                        Only dest is modified; source is read-only
                        --delete  Also remove files on dest that are not in source
                        devbox sync ./project mybox:/home/ec2-user/project
                        devbox sync mybox:/home/ec2-user/project ./project
  exec [-i key] [-s] [-t] <id|name> -- <command>
                      Run a one-off command on a running box
                        -s  Run as a shell snippet via sh -lc (pipes, &&, cd)
                        -t  Allocate a pseudo-TTY (for sudo / interactive commands)
  forward <id|name> <port>
                      Forward a port from a box
`)
}

func helpSnapshot() {
	fmt.Fprintf(os.Stderr, `Usage: devbox snapshot [subcommand] [args]

A snapshot is a saved disk image of a box; restore one with create --from.

  snapshot                              List all snapshots
  snapshot create <id|name> <name>      Create a snapshot of a box
  snapshot ls <amiId|name>              Show details for a snapshot
  snapshot delete <amiId|name>          Delete a snapshot
`)
}

func helpTemplate() {
	fmt.Fprintf(os.Stderr, `Usage: devbox template [subcommand] [args]

Templates let you create boxes preloaded with libs, tools, and other setup.

  template                              List available templates
  template new <templateName> [command string]  Create a template with optional startup command
  template delete <templateName>                Delete a template
  template rename <templateName> <new-templateName>     Rename a template
  template search <query>               Search templates by name ( returns partial matches )
`)
}

func helpIdleStop() {
	fmt.Fprintf(os.Stderr, `Usage: devbox idle-stop <subcommand> <id|name> [args]

  idle-stop set <id|name> <minutes>     Stop the box after inactivity
  idle-stop show <id|name>              Show idle-stop settings for a box
  idle-stop update <id|name> <minutes>  Update idle-stop timeout
  idle-stop delete <id|name>            Remove idle-stop from a box
`)
}

func helpGitSync() {
	fmt.Fprintf(os.Stderr, `Usage: devbox git-sync <id|name>

  git-sync <id|name>  Toggle GitHub SSH access for a box: adds the local key
                      to ssh-agent and enables agent forwarding (-A) in the
                      box's SSH config; run again to undo both.
`)
}

func helpCommand(args []string) {
	if len(args) == 0 {
		usage()
		return
	}

	switch args[0] {
	case "create":
		helpCreate()
	case "box":
		helpBox()
	case "ssh":
		helpSSH()
	case "snapshot":
		helpSnapshot()
	case "template":
		helpTemplate()
	case "idle-stop":
		helpIdleStop()
	case "git-sync":
		helpGitSync()
	default:
		fmt.Fprintf(os.Stderr, "devbox: unknown help topic %q\n\n", args[0])
		fmt.Fprintf(os.Stderr, "Topics: %s\n", helpTopics)
		os.Exit(1)
	}
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
	case "help":
		helpCommand(args)
		os.Exit(0)
	case "-h", "--help":
		usage()
		os.Exit(0)
	case "version":
		cmd.Version(args)
	case "update":
		cmd.Update(args)
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
	case "template":
		cmd.Template(args)
	case "idle-stop":
		cmd.IdleRouter(args)
	case "git-sync":
		cmd.GitSync(args)
	default:
		fmt.Fprintf(os.Stderr, "devbox: unknown command %q\n\n", command)
		usage()
		os.Exit(1)
	}
}
