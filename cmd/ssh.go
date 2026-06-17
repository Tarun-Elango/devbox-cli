package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"devbox-cli/internal/api"
	"devbox-cli/service"
)

const (
	devboxReadyPath    = "/var/lib/devbox/ready"
	devboxReadyMessage = "the user data script is completed"
)

var execCommand = exec.Command

// defaultKeyPath returns the path to the user's default SSH private key,
// trying id_ed25519 then id_rsa under ~/.ssh. Returns "" if none found.
func defaultKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, name := range []string{"id_ed25519", "id_rsa"} {
		p := filepath.Join(home, ".ssh", name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
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

// ssh ec2-user@ip 'test "$(cat /var/lib/devbox/ready 2>/dev/null)" = "the user data script is completed"'; echo "exit code: $?"
// checkDevboxReady runs one SSH probe for the user-data ready marker.
func checkDevboxReady(sshBin, identity, user, host, portArg string) (bool, error) {
	target := fmt.Sprintf("%s@%s", user, host)
	probe := fmt.Sprintf(`test "$(cat %s 2>/dev/null)" = %q`, devboxReadyPath, devboxReadyMessage)
	argv := append([]string{sshBin}, sshBaseArgs(identity, portArg)...)
	argv = append(argv,
		"-o", "BatchMode=yes",
		"-o", "LogLevel=ERROR",
		target,
		probe,
	)
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

	ready, err := checkDevboxReady(sshBin, *identity, *user, b.PublicIP, portArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh: readiness probe failed (%v); attempting SSH anyway\n", err)
	} else if !ready {
		fmt.Fprintln(os.Stderr, "ssh: devbox is not ready yet — try again in a minute")
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
