package main

import (
	"fmt"
	"os"

	"outpost-cli/cmd"
)

const helpTopics = "create, box, ssh, snapshot, template, idle-stop, git-sync, budget"

var commandHelpTopics = map[string]string{
	"create":    "create",
	"box":       "box",
	"ls":        "box",
	"status":    "box",
	"rename":    "box",
	"resize":    "box",
	"upgrade":   "box",
	"stop":      "box",
	"start":     "box",
	"restart":   "box",
	"reboot":    "box",
	"delete":    "box",
	"ssh":       "ssh",
	"cp":        "ssh",
	"sync":      "ssh",
	"exec":      "ssh",
	"forward":   "ssh",
	"snapshot":  "snapshot",
	"template":  "template",
	"idle-stop": "idle-stop",
	"git-sync":  "git-sync",
	"budget":    "budget",
	"cost":      "budget",
	"bill":      "budget",
}

func isHelpFlag(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

// resolveHelpTopic handles help anywhere in the invocation, e.g.
// "outpost help box", "outpost box help", or "outpost snapshot create help".
func resolveHelpTopic(command string, args []string) (topic string, showGeneral bool, ok bool) {
	if command == "help" || isHelpFlag(command) {
		for _, arg := range args {
			if !isHelpFlag(arg) {
				return arg, false, true
			}
		}
		return "", true, true
	}

	for _, arg := range args {
		// if the argument is a help flag, return the topic from the map
		if isHelpFlag(arg) {
			if t, exists := commandHelpTopics[command]; exists {
				return t, false, true
			}
			return "", true, true
		}
	}

	return "", false, false
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: outpost <command> [args]

Commands:
  help, -h, --help    Show this help message
  help <topic>        Show help for a topic (%s)

  version             Show the outpost CLI version
  update              Check for a newer release and install it (asks for confirmation)
  uninstall           Remove outpost, local data, and PATH entries (asks for confirmation)

  setup               Configure AWS credentials and region (stored in ~/.outpost/config.json)
  clear-creds         Clear saved AWS credentials from ~/.outpost/config.json
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
                        outpost cp ./main.go mybox:/home/ec2-user/app/
                        outpost cp mybox:/home/ec2-user/app/main.go ./
  sync [-i key] [--delete] <source> <dest>
                      Sync directories via rsync over SSH (one local path, one box:/path)
                        Only dest is modified; source is read-only.
                        --delete also removes files on dest that are not in source
                        outpost sync ./project mybox:/home/ec2-user/project
                        outpost sync mybox:/home/ec2-user/project ./project
  exec [-i key] [-s] [-t] <id|name> -- <command>
                      Run a one-off command on a running box
                        -s  Run as a shell snippet via sh -lc (pipes, &&, cd)
                        -t  Allocate a pseudo-TTY (for sudo / interactive commands)
  forward <id|name> <port>
                      Forward a port from a box

  snapshot [ls] [<amiId|name>]          List all snapshots, or show details for one
  snapshot create <id|name> <name>      Create a snapshot of a box
  snapshot delete <amiId|name>          Delete a snapshot

  template [ls]                         List available templates
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

  budget [ls] [--refresh]
                      List AWS account budgets (name, period, limit, spend,
                      forecast, %% of budget)
                      Results are cached under ~/.outpost/ for 12h.
  budget create <name> <limit> <email>
                      Create a monthly cost budget for all AWS services.
                      Alerts at 85%% actual, 100%% actual, and 100%% forecasted spend.
  budget update <name>  Interactively update name, limit, or alert email
  budget delete <name>  Delete a budget by exact name (quote names with spaces)
`, helpTopics)
}

func helpCreate() {
	fmt.Fprintf(os.Stderr, `Usage: outpost create <name> [--template <templateName>...] [--from <amiId|name>]

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
	fmt.Fprintf(os.Stderr, `Usage: outpost <box-command> <id|name> [args]

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
	fmt.Fprintf(os.Stderr, `Usage: outpost <ssh-command> [args]

  ssh [-i key] <id|name> [-- <ssh-option>...]
                      Open an SSH session to a box
                        -i  Path to SSH private key (default: ~/.ssh/id_ed25519)
                        --  Pass native ssh options before the target (e.g. -v, -A,
                            -L 8080:localhost:8080); for one-off remote commands use exec
  cp [-i key] <source> <dest>
                      Copy a file to or from a box
                        outpost cp ./main.go mybox:/home/ec2-user/app/
                        outpost cp mybox:/home/ec2-user/app/main.go ./
  sync [-i key] [--delete] <source> <dest>
                      Sync directories via rsync over SSH (one local path, one box:/path)
                        Only dest is modified; source is read-only
                        --delete  Also remove files on dest that are not in source
                        outpost sync ./project mybox:/home/ec2-user/project
                        outpost sync mybox:/home/ec2-user/project ./project
  exec [-i key] [-s] [-t] <id|name> -- <command>
                      Run a one-off command on a running box
                        -s  Run as a shell snippet via sh -lc (pipes, &&, cd)
                        -t  Allocate a pseudo-TTY (for sudo / interactive commands)
  forward <id|name> <port>
                      Forward a port from a box
`)
}

func helpSnapshot() {
	fmt.Fprintf(os.Stderr, `Usage: outpost snapshot [subcommand] [args]

A snapshot is a saved disk image of a box; restore one with create --from.

  snapshot [ls] [<amiId|name>]          List all snapshots, or show details for one
  snapshot create <id|name> <name>      Create a snapshot of a box
  snapshot delete <amiId|name>          Delete a snapshot
`)
}

func helpTemplate() {
	fmt.Fprintf(os.Stderr, `Usage: outpost template [subcommand] [args]

Templates let you create boxes preloaded with libs, tools, and other setup.

  template [ls]                         List available templates
  template new <templateName> [command string]  Create a template with optional startup command
  template delete <templateName>                Delete a template
  template rename <templateName> <new-templateName>     Rename a template
  template search <query>               Search templates by name ( returns partial matches )
`)
}

func helpIdleStop() {
	fmt.Fprintf(os.Stderr, `Usage: outpost idle-stop <subcommand> <id|name> [args]

  idle-stop set <id|name> <minutes>     Stop the box after inactivity
  idle-stop show <id|name>              Show idle-stop settings for a box
  idle-stop update <id|name> <minutes>  Update idle-stop timeout
  idle-stop delete <id|name>            Remove idle-stop from a box
`)
}

func helpGitSync() {
	fmt.Fprintf(os.Stderr, `Usage: outpost git-sync <id|name>

  git-sync <id|name>  Toggle GitHub SSH access for a box: adds the local key
                      to ssh-agent and enables agent forwarding (-A) in the
                      box's SSH config; run again to undo both.
`)
}

func helpBudget() {
	fmt.Fprintf(os.Stderr, `Usage: outpost budget [ls] [--refresh] | create <name> <limit> <email> | update <name> | delete <name>

  budget                List all AWS account budgets
  budget ls             Same as above
  budget --refresh      Bypass the local cache and refetch from AWS
  budget create <name> <limit> <email>
                        Create a monthly cost budget for all AWS services.
                        Alerts at 85%% actual, 100%% actual, and 100%% forecasted spend.
  budget update <name>  Interactively update name, limit, or alert email
                        (Enter keeps each current value)
  budget delete <name>  Delete a budget by exact name (quote names with spaces)

Budgets require AWSBudgetsActionsWithAWSResourceControlAccess permission policy to the IAM user
Results are cached under ~/.outpost/ for 12h since repeated calls aren't necessary (Budgets API is free).
`)
}

func showHelpTopic(topic string) {
	switch topic {
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
	case "budget", "cost", "bill":
		helpBudget()
	default:
		fmt.Fprintf(os.Stderr, "outpost: unknown help topic %q\n\n", topic)
		fmt.Fprintf(os.Stderr, "Topics: %s\n", helpTopics)
		os.Exit(1)
	}
}

func main() {
	// if len(os.Args) < 2 || os.Args[1] != "uninstall" {
	// 	// backup.MaybeDaily()
	// }
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	if topic, general, ok := resolveHelpTopic(command, args); ok {
		if general {
			usage()
		} else {
			showHelpTopic(topic)
		}
		os.Exit(0)
	}

	switch command {
	case "version":
		cmd.Version(args)
	case "update":
		cmd.Update(args)
	case "uninstall":
		cmd.Uninstall(args)
	case "setup":
		cmd.Setup(args)
	case "clear-creds":
		cmd.ClearCreds(args)
	case "health":
		cmd.Health(args)
	case "budget":
		cmd.Budget(args)
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
		fmt.Fprintf(os.Stderr, "outpost: unknown command %q\n\n", command)
		usage()
		os.Exit(1)
	}
}

//touch 2
