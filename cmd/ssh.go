package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"devbox-cli/internal/api"
	"devbox-cli/service"
)

const (
	devboxReadyPath         = "/var/lib/devbox/ready"
	devboxReadyMessage      = "the user data script is completed"
	devboxReadyPollInterval = 5 * time.Second
)

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

// helper: ensureEd25519Key runs ssh-keygen to create ~/.ssh/id_ed25519 when the user confirms.
func ensureEd25519Key() error {
	priv, _, err := ed25519KeyPaths()
	if err != nil {
		return err
	}

	sshKeygen, err := exec.LookPath("ssh-keygen") // look for ssh-keygen binary in PATH
	if err != nil {
		return fmt.Errorf("ssh-keygen not found in PATH")
	}

	if err := os.MkdirAll(filepath.Dir(priv), 0o700); err != nil { // create the ~/.ssh directory if it doesn't exist
		return fmt.Errorf("create ~/.ssh: %w", err)
	}

	cmd := exec.Command(sshKeygen, "-t", "ed25519", "-f", priv) // create the ed25519 key pair
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen failed: %w", err)
	}
	return nil
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

// SshStatusResponse is returned by GET /v2/boxes/{id}/ssh-status.
type SshStatusResponse struct {
	Ready    bool `json:"ready"`
	Instance *Box `json:"instance"`
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

// checkDevboxReady runs one SSH probe for the user-data ready marker.
// Returns (true, nil) when ready, (false, nil) when SSH works but provisioning
// is incomplete, and (false, err) when SSH is not reachable yet.
func checkDevboxReady(sshBin, identity, user, host, portArg string) (bool, error) {
	target := fmt.Sprintf("%s@%s", user, host)
	probe := fmt.Sprintf(`test "$(cat %s 2>/dev/null)" = %q`, devboxReadyPath, devboxReadyMessage)
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

// waitForDevboxReady polls until the user-data ready marker is present or the user cancels.
func waitForDevboxReady(sshBin, identity, user, host, portArg string) error {
	sigCh := make(chan os.Signal, 1)   // channel to receive signals
	signal.Notify(sigCh, os.Interrupt) // notify when interrupt signal is received
	defer signal.Stop(sigCh)           // stop listening for signals

	for {
		ready, err := checkDevboxReady(sshBin, identity, user, host, portArg)
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
		case <-time.After(devboxReadyPollInterval):
		case <-sigCh: // if interrupt signal is received, cancel the operation
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("cancelled")
		}
	}
}

// SSH checks EC2 health and the devbox ready marker, then execs ssh.
func SSH(args []string) {
	if TestMode {
		fmt.Println("[test] ssh: done")
		return
	}
	fs := flag.NewFlagSet("ssh", flag.ExitOnError)
	user := fs.String("u", "ec2-user", "SSH username")
	port := fs.Int("p", 22, "SSH port")
	identity := fs.String("i", defaultKeyPath(), "path to SSH private key") // ssh picks the ssh private key for creating the connection
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: devbox ssh -v -i [-u user] [-p port] [-i identity] <id> [-- ssh-args...]")
		fs.PrintDefaults()
	}

	// Split args on "--" to allow passing raw flags to ssh.
	var extra []string
	for i, a := range args {
		if a == "--" {
			extra = args[i+1:]
			args = args[:i]
			break
		}
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	id := fs.Arg(0)

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var status SshStatusResponse
	if mode == "local" {
		rt := mustOpenRuntime()
		sshStatus, err := rt.GetSshStatus(id, service.LocalUserID)
		closeErr := rt.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
			os.Exit(1)
		}
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", closeErr)
			os.Exit(1)
		}
		status.Ready = sshStatus.Ready
		if sshStatus.Instance != nil {

			box := instancesToBoxes([]*service.Instance{sshStatus.Instance})[0]
			status.Instance = &box
		}
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Get("/v2/boxes/" + id + "/ssh-status")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
			os.Exit(1)
		}
		if err := api.CheckStatus(resp); err != nil {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
			os.Exit(1)
		}
		if err := api.DecodeJSON(resp, &status); err != nil {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
			os.Exit(1)
		}
	}

	if !status.Ready {
		fmt.Fprintln(os.Stderr, "ssh: box is not ready yet (EC2 status checks still pending)")
		os.Exit(1)
	}
	if status.Instance == nil {
		fmt.Fprintln(os.Stderr, "ssh: server reported ready but returned no instance details, try the command again in a few minutes.")
		os.Exit(1)
	}

	b := *status.Instance
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

	target := fmt.Sprintf("%s@%s", *user, b.PublicIP)
	portArg := fmt.Sprintf("%d", *port)

	if err := waitForDevboxReady(sshBin, *identity, *user, b.PublicIP, portArg); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Connecting to %s (box %s)...\n", target, id)

	argv := append([]string{sshBin}, sshBaseArgs(*identity, portArg)...) // create ssh command
	argv = append(argv, target)
	argv = append(argv, extra...)

	if err := syscall.Exec(sshBin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: exec failed: %v\n", err)
		os.Exit(1)
	}
}

/*
ssh flowchart
flowchart TD
    A[Parse flags + box id] --> B{mode}
    B -->|local| C[GetSshStatus + manual map to Box]
    B -->|remote| D[GET /v2/boxes/id/ssh-status]
    C --> E[Unified validation]
    D --> E
    E --> F{ready?}
    F -->|no| X[exit]
    F -->|yes| G{instance + IP + running?}
    G -->|no| X
    G -->|yes| H[checkDevboxReady probe]
    H --> I[syscall.Exec ssh]

*/
