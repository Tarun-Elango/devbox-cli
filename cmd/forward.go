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
	defer func() { _ = l.Close() }()
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
		rt := mustOpenRuntime()
		resp, err := rt.ForwardPort(id, port, service.LocalUserID)
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
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		statusResp, err := client.Get("/v2/boxes/" + id + "/ssh-status")
		if err != nil {
			fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
			os.Exit(1)
		}
		if err := api.CheckStatus(statusResp); err != nil {
			fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
			os.Exit(1)
		}
		var status SshStatusResponse
		if err := api.DecodeJSON(statusResp, &status); err != nil {
			fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
			os.Exit(1)
		}
		if !status.Ready {
			fmt.Fprintln(os.Stderr, "forward: box is not ready yet (EC2 status checks still pending)")
			os.Exit(1)
		}
		if status.Instance == nil {
			fmt.Fprintln(os.Stderr, "forward: server reported ready but returned no instance details, try the command again in a few minutes.")
			os.Exit(1)
		}
		if status.Instance.PublicIP == "" {
			fmt.Fprintln(os.Stderr, "forward: box has no IP address (is it running?)")
			os.Exit(1)
		}
		if status.Instance.Status != "running" {
			fmt.Fprintf(os.Stderr, "forward: box is %s, not running\n", status.Instance.Status)
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

	sshBin, err := exec.LookPath("ssh") // look for ssh binary in PATH
	if err != nil {
		fmt.Fprintln(os.Stderr, "forward: ssh binary not found in PATH")
		os.Exit(1)
	}

	// we also check if user-data script is completed
	identity := defaultKeyPath()
	ready, err := checkDevboxReady(sshBin, identity, result.User, result.Host, "22")
	if err != nil {
		fmt.Fprintf(os.Stderr, "forward: SSH is not reachable yet (%v) — the box may still be starting, try again in a moment\n", err)
		os.Exit(1)
	}
	if !ready {
		fmt.Fprintln(os.Stderr, "forward: devbox is not ready yet — try again in a minute")
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
	target := fmt.Sprintf("%s@%s", result.User, result.Host)

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
