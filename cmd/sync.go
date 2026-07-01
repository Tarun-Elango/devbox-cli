package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"devbox-cli/helper"
)

func syncRemoteShell(identity, portArg string) string {
	parts := []string{
		"ssh",
		"-p", portArg,
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	if identity != "" {
		parts = append([]string{"ssh", "-i", shellQuote(identity)}, parts[1:]...)
	}
	return strings.Join(parts, " ") // return sync remote shell command
}

// build the rsync command
func buildRsyncArgs(identity string, transfer cpTransfer, user, host, portArg string, deleteExtra bool) []string {
	remote := fmt.Sprintf("%s@%s:%s", user, host, transfer.Remote)
	argv := []string{
		"-az",
		"-e", syncRemoteShell(identity, portArg),
	}
	if deleteExtra {
		argv = append(argv, "--delete")
	}
	if transfer.Upload {
		return append(argv, transfer.Local, remote)
	}
	return append(argv, remote, transfer.Local)
}

// Sync synchronizes files or directories between the local machine and a devbox using rsync.
func Sync(args []string) {

	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: devbox sync [-i key] [--delete] <source> <dest>")
		fmt.Fprintln(os.Stderr, "examples:")
		fmt.Fprintln(os.Stderr, "  devbox sync ./project mybox:/home/ec2-user/project")
		fmt.Fprintln(os.Stderr, "  devbox sync mybox:/home/ec2-user/project ./project")
	}

	parsed, err := helper.ParseSyncCommandArgs(args, cpDefaultKeyPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		fs.Usage()
		os.Exit(1)
	}

	transfer, err := parseCPTransfer(parsed.Source, parsed.Dest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}

	status, err := cpStatusForBox(transfer.BoxRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}
	if !status.Ready {
		fmt.Fprintln(os.Stderr, "sync: box is not ready yet.")
		os.Exit(1)
	}
	if status.Instance == nil {
		fmt.Fprintln(os.Stderr, "sync: server reported ready but returned no instance details, try the command again in a few minutes.")
		os.Exit(1)
	}
	if status.Instance.PublicIP == "" {
		fmt.Fprintln(os.Stderr, "sync: box has no IP address (is it running?)")
		os.Exit(1)
	}
	if status.Instance.Status != "running" {
		fmt.Fprintf(os.Stderr, "sync: box is %s, not running\n", status.Instance.Status)
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync: ssh binary not found in PATH")
		os.Exit(1)
	}

	rsyncBin, err := exec.LookPath("rsync")
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync: rsync binary not found in PATH")
		os.Exit(1)
	}

	if err := waitForDevboxReady(sshBin, parsed.Identity, cpDefaultSSHUser, status.Instance.PublicIP, cpDefaultSSHPort); err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}

	// build the rsync command
	argv := buildRsyncArgs(parsed.Identity, transfer, cpDefaultSSHUser, status.Instance.PublicIP, cpDefaultSSHPort, parsed.DeleteExtra)
	cmd := exec.Command(rsyncBin, argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "sync: rsync failed: %v\n", err)
		os.Exit(1)
	}
}
