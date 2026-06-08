package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"

	"devbox-cli/internal/api"
	"devbox-cli/service"
)

// findFreePort asks the OS for an available TCP port on localhost.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Forward asks the server for connection details, then establishes an SSH
// local port-forward so that localhost:<localPort> proxies to the box's
// <remotePort>.  Blocks until the user presses Ctrl-C.
// Usage: devbox forward <id> <port>
func Forward(args []string) {
	if TestMode {
		fmt.Println("[test] forward: done")
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox forward <id> <port>")
		os.Exit(1)
	}
	id := args[0]
	port := args[1]

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var result service.PortForwardResponse
	if mode == "local" {
		resp, err := service.ForwardPort(id, port, service.LocalUserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
			os.Exit(1)
		}
		result = *resp
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		body := map[string]string{"port": port}
		resp, err := client.Post("/v1/boxes/"+id+"/ports", body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
			os.Exit(1)
		}
		if err := api.CheckStatus(resp); err != nil {
			fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
			os.Exit(1)
		}
		if err := api.DecodeJSON(resp, &result); err != nil { // add to result var
			fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Find a free local port to forward to.  We can't assume the requested port is free,
	// and we don't want to require the user to specify a local port at all.
	localPort, err := findFreePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "forward: could not find a free local port: %v\n", err)
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "forward: ssh binary not found in PATH")
		os.Exit(1)
	}

	bindSpec := fmt.Sprintf("%s:localhost:%s", strconv.Itoa(localPort), result.RemotePort)
	target := fmt.Sprintf("%s@%s", result.User, result.Host)

	argv := []string{
		sshBin,
		"-N",
		"-L", bindSpec,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=15",
	}
	if key := defaultKeyPath(); key != "" {
		argv = append(argv, "-i", key)
	}
	argv = append(argv, target)

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "forward: failed to start ssh: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Forwarded: http://localhost:%d  (Ctrl+C to stop)\n", localPort)

	if err := cmd.Wait(); err != nil {
		// ssh exits non-zero when killed by a signal (Ctrl+C) — that's expected.
		if cmd.ProcessState != nil && !cmd.ProcessState.Success() {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "forward: ssh exited: %v\n", err)
		os.Exit(1)
	}
}
