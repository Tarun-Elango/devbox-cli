package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"outpost-cli/helper"
	"outpost-cli/service"
)

const (
	outpostReadyPath         = "/var/lib/outpost/ready"
	outpostReadyMessage      = "the user data script is completed"
	outpostReadyPollInterval = 5 * time.Second
	defaultSSHUser           = "ec2-user"
	defaultSSHPort           = "22"
)

// go's way to run shell commands, code needs to call
var execCommand = exec.Command

// helper: ed25519KeyPaths returns paths to ~/.ssh/id_ed25519 and ~/.ssh/id_ed25519.pub.
func ed25519KeyPaths() (privateKey, publicKey string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("could not determine home directory: %w", err)
	}
	sshDir := filepath.Join(home, ".ssh")
	return filepath.Join(sshDir, "id_ed25519"), filepath.Join(sshDir, "id_ed25519.pub"), nil
}

// defaultKeyPath returns ~/.ssh/id_ed25519 if it exists, otherwise "".
func defaultKeyPath() string {
	priv, _, err := ed25519KeyPaths()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(priv); err == nil {
		return priv
	}
	return ""
}

func sshBaseArgs(identity, portArg string) []string {
	argv := []string{
		"-p", portArg,
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new", // TODO: StrictHostKeyChecking=yes plus managing known_hosts
	}
	if identity != "" {
		argv = append([]string{"-i", identity}, argv...)
	}
	return argv
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

// buildExecRemoteCommand formats remote argv for SSH exec.
// When throughShell is true, remoteCommand is joined into one shell snippet
// executed via sh -lc, so original argument boundaries are not preserved.
// Otherwise each argument is shell-quoted individually and passed as-is.
func buildExecRemoteCommand(remoteCommand []string, throughShell bool) []string {
	if throughShell {
		return []string{"sh -lc " + shellQuote(strings.Join(remoteCommand, " "))}
	}

	quoted := make([]string, 0, len(remoteCommand))
	for _, arg := range remoteCommand {
		quoted = append(quoted, shellQuote(arg))
	}
	return []string{strings.Join(quoted, " ")}
}

// checkoutpostReady runs one SSH probe for the user-data ready marker.
// Returns (true, nil) when ready, (false, nil) when SSH works but provisioning
// is incomplete, and (false, err) when SSH is not reachable yet.
func checkoutpostReady(sshBin, identity, user, host, portArg string) (bool, error) {
	target := fmt.Sprintf("%s@%s", user, host)
	probe := fmt.Sprintf(`test "$(cat %s 2>/dev/null)" = %q`, outpostReadyPath, outpostReadyMessage)
	argv := []string{sshBin,
		"-p", portArg,
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "BatchMode=yes",
		"-o", "LogLevel=ERROR",
	}
	if identity != "" {
		argv = append([]string{sshBin, "-i", identity}, argv[1:]...)
	}
	argv = append(argv, target, probe)

	err := execCommand(argv[0], argv[1:]...).Run()
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}

	return false, err
}

// waitForoutpostReady polls until the user-data ready marker is present or the user cancels.
func waitForoutpostReady(sshBin, identity, user, host, portArg string) error {
	sigCh := make(chan os.Signal, 1)   // channel to receive signals
	signal.Notify(sigCh, os.Interrupt) // notify when interrupt signal is received
	defer signal.Stop(sigCh)           // stop listening for signals

	for {
		ready, err := checkoutpostReady(sshBin, identity, user, host, portArg)
		if ready {
			return nil
		}

		var msg string
		if err != nil {
			msg = "waiting for SSH to become reachable, might take a moment. Press Ctrl+C to stop waiting."
		} else {
			msg = "waiting for templates to be installed, might take a moment. Press Ctrl+C to stop waiting."
		}
		fmt.Fprintf(os.Stderr, "ssh: %s\n", msg)

		select {
		case <-time.After(outpostReadyPollInterval):
		case <-sigCh: // if interrupt signal is received, cancel the operation
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("cancelled")
		}
	}
}

// SSH checks EC2 health and the outpost ready marker, then execs ssh.
// Arguments after "--" are forwarded as native ssh options (e.g. -v, -A,
// -L 8080:localhost:8080) rather than as a remote command; use "outpost exec"
// to run a one-off remote command instead.
func SSH(args []string) {

	usage := func() {
		fmt.Fprintln(os.Stderr, "usage: outpost ssh [-i key] <id|name> [-- <ssh-option>...]")
	}

	parsed, err := helper.ParseSSHCommandArgs(args, defaultKeyPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		usage()
		os.Exit(1)
	}
	ref := parsed.Ref

	var targetLabel string

	rt := helper.MustOpenRuntime()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		_ = rt.Close()
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(1)
	}
	targetLabel = fmt.Sprintf("%s (%s)", target.Name, target.ID)
	inst, err := rt.GetInstance(target.ID, service.LocalUserID)
	closeErr := rt.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", closeErr)
		os.Exit(1)
	}

	b := instancesToBoxes([]*service.Instance{inst})[0]
	if b.PublicIP == "" {
		fmt.Fprintln(os.Stderr, "ssh: box has no IP address (is it running?)")
		os.Exit(1)
	}
	if b.Status != "running" {
		fmt.Fprintf(os.Stderr, "ssh: box is %s, not running\n", b.Status)
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ssh: ssh binary not found in PATH")
		os.Exit(1)
	}

	sshTarget := fmt.Sprintf("%s@%s", defaultSSHUser, b.PublicIP)

	if err := waitForoutpostReady(sshBin, parsed.Identity, defaultSSHUser, b.PublicIP, defaultSSHPort); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Connecting to %s (box %s)...\n", sshTarget, targetLabel)

	argv := append([]string{sshBin}, sshBaseArgs(parsed.Identity, defaultSSHPort)...) // create ssh command
	argv = append(argv, parsed.SSHOptions...)                                         // user-supplied ssh flags (-v, -A, -L, -o, ...)
	argv = append(argv, sshTarget)

	if err := syscall.Exec(sshBin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: exec failed: %v\n", err)
		os.Exit(1)
	}
}

// Exec runs a one-off command on a running box over SSH.
func Exec(args []string) {

	fs := flag.NewFlagSet("exec", flag.ExitOnError)
	identity := fs.String("i", defaultKeyPath(), "path to SSH private key") // something like ~/.ssh/id_ed25519
	throughShell := fs.Bool("s", false, "run as a shell snippet via sh -lc (pipes, &&, cd); joins arguments and does not preserve per-arg boundaries (see buildExecRemoteCommand)")
	allocateTTY := fs.Bool("t", false, "allocate a pseudo-TTY")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: outpost exec [-i identity] [-s] [-t] <id|name> -- <command>")
		fs.PrintDefaults()
	}

	var remoteCommand []string
	for i, a := range args {
		if a == "--" {
			remoteCommand = args[i+1:]
			args = args[:i]
			break
		}
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	identityPath := helper.StripSurroundingQuotes(*identity)
	if fs.NArg() != 1 || len(remoteCommand) == 0 {
		fs.Usage()
		os.Exit(1)
	}
	ref := fs.Arg(0)

	rt := helper.MustOpenRuntime()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		_ = rt.Close()
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}
	inst, err := rt.GetInstance(target.ID, service.LocalUserID)
	closeErr := rt.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", closeErr)
		os.Exit(1)
	}

	b := instancesToBoxes([]*service.Instance{inst})[0]
	if b.PublicIP == "" {
		fmt.Fprintln(os.Stderr, "exec: box has no IP address (is it running?)")
		os.Exit(1)
	}
	if b.Status != "running" {
		fmt.Fprintf(os.Stderr, "exec: box is %s, not running\n", b.Status)
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "exec: ssh binary not found in PATH")
		os.Exit(1)
	}

	if err := waitForoutpostReady(sshBin, identityPath, defaultSSHUser, b.PublicIP, defaultSSHPort); err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}

	sshTarget := fmt.Sprintf("%s@%s", defaultSSHUser, b.PublicIP)
	argv := sshBaseArgs(identityPath, defaultSSHPort)
	if *allocateTTY {
		argv = append(argv, "-t")
	}
	argv = append(argv, sshTarget)
	argv = append(argv, buildExecRemoteCommand(remoteCommand, *throughShell)...)

	fmt.Fprintln(os.Stderr, "exec: trying command...")

	cmd := execCommand(sshBin, argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "exec: ssh failed: %v\n", err)
		os.Exit(1)
	}
}

/*
ssh flowchart
flowchart TD
    A[Parse flags + box id] --> B[GetInstance + map to Box]
    B --> C{IP + running?}
    C -->|no| X[exit]
    C -->|yes| D[waitForoutpostReady probe]
    D --> E[syscall.Exec ssh]

*/
