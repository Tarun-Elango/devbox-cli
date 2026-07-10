package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"outpost-cli/helper"
	"outpost-cli/service"
)

// findFreePort asks the OS for an available TCP port on localhost.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Forward asks the server for connection details, then establishes an SSH
// local port-forward so that localhost:<localPort> proxies to the box's
// <remotePort>.  Blocks until the user presses Ctrl-C.
// Usage: outpost forward <id|name> <port>
func Forward(args []string) {

	ref, port := helper.ParseForwardArgs(args, "usage: outpost forward <id|name> <port>")

	var result service.PortForwardResponse

	rt := helper.MustOpenRuntime()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		_ = rt.Close()
		fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
		os.Exit(1)
	}
	resp, err := rt.ForwardPort(target.ID, port, service.LocalUserID)
	closeErr := rt.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "forward failed: %v\n", closeErr)
		os.Exit(1)
	}
	result = *resp

	sshBin, err := exec.LookPath("ssh") // look for ssh binary in PATH
	if err != nil {
		fmt.Fprintln(os.Stderr, "forward: ssh binary not found in PATH")
		os.Exit(1)
	}

	// we also check if user-data script is completed
	identity := defaultKeyPath()
	ready, err := checkoutpostReady(sshBin, identity, result.User, result.Host, "22")
	if err != nil {
		fmt.Fprintf(os.Stderr, "forward: SSH is not reachable yet (%v) — the box may still be starting, try again in a moment\n", err)
		os.Exit(1)
	}
	if !ready {
		fmt.Fprintln(os.Stderr, "forward: outpost is not ready yet — try again in a minute")
		os.Exit(1)
	}

	// Find a free local port to forward to.  We can't assume the requested port is free,
	// and we don't want to require the user to specify a local port at all.
	localPort, err := findFreePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "forward: could not find a free local port: %v\n", err)
		os.Exit(1)
	}

	bindSpec := fmt.Sprintf("%s:localhost:%s", strconv.Itoa(localPort), result.RemotePort)
	sshTarget := fmt.Sprintf("%s@%s", result.User, result.Host)

	argv := []string{
		sshBin,
		"-N",
		"-L", bindSpec,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=15",
	}
	if identity != "" {
		argv = append(argv, "-i", identity)
	}
	argv = append(argv, sshTarget)

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "forward: failed to start ssh: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Forwarded: http://localhost:%d  (Ctrl+C to stop)\n", localPort)

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "forward: ssh exited: %v\n", err)
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "forward: ssh exited: %v\n", err)
		os.Exit(1)
	}
}
