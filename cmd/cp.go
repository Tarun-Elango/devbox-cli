package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"devbox-cli/helper"
	"devbox-cli/service"
)

const (
	cpDefaultSSHUser = "ec2-user"
	cpDefaultSSHPort = "22"
)

type cpLocation struct {
	Raw    string
	Remote bool
	Box    string
	Path   string
}

type cpTransfer struct {
	Source cpLocation
	Dest   cpLocation
	Upload bool
	BoxRef string
	Remote string
	Local  string
}

type cpStatusResponse struct {
	Ready    bool `json:"ready"`
	Instance *Box `json:"instance"`
}

// the default key path is ~/.ssh/id_ed25519
func cpDefaultKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	priv := filepath.Join(home, ".ssh", "id_ed25519")
	if _, err := os.Stat(priv); err == nil {
		return priv
	}
	return ""
}

func parseCPLocation(raw string) (cpLocation, error) {
	raw = helper.StripSurroundingQuotes(strings.TrimSpace(raw))
	if raw == "" {
		return cpLocation{}, fmt.Errorf("path is required")
	}

	box, path, ok := strings.Cut(raw, ":")
	if !ok {
		return cpLocation{Raw: raw, Path: raw}, nil
	}
	if strings.TrimSpace(box) == "" {
		return cpLocation{}, fmt.Errorf("remote path %q is missing a box name or id", raw)
	}
	if strings.TrimSpace(path) == "" {
		return cpLocation{}, fmt.Errorf("remote path %q is missing a path", raw)
	}

	return cpLocation{
		Raw:    raw,
		Remote: true,
		Box:    strings.TrimSpace(box),
		Path:   path,
	}, nil
}

// parseCPTransfer parses the source and destination arguments and returns a cpTransfer struct
func parseCPTransfer(sourceArg, destArg string) (cpTransfer, error) {
	source, err := parseCPLocation(sourceArg)
	if err != nil {
		return cpTransfer{}, err
	}
	dest, err := parseCPLocation(destArg)
	if err != nil {
		return cpTransfer{}, err
	}

	switch {
	case source.Remote && dest.Remote:
		return cpTransfer{}, fmt.Errorf("copying between two boxes is not supported")
	case !source.Remote && !dest.Remote:
		return cpTransfer{}, fmt.Errorf("one path must be remote in the form <id|name>:/path")
	case dest.Remote:
		return cpTransfer{
			Source: source,
			Dest:   dest,
			Upload: true,
			BoxRef: dest.Box,
			Remote: dest.Path,
			Local:  source.Path,
		}, nil
	default:
		return cpTransfer{
			Source: source,
			Dest:   dest,
			Upload: false,
			BoxRef: source.Box,
			Remote: source.Path,
			Local:  dest.Path,
		}, nil
	}
}

func cpSCPBaseArgs(identity, portArg string) []string {
	argv := []string{
		"-P", portArg,
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	if identity != "" {
		argv = append([]string{"-i", identity}, argv...)
	}
	return argv
}

// build the scp command
func buildSCPArgs(identity string, transfer cpTransfer, user, host, portArg string) []string {
	remote := fmt.Sprintf("%s@%s:%s", user, host, transfer.Remote)
	argv := cpSCPBaseArgs(identity, portArg)
	if transfer.Upload {
		return append(argv, transfer.Local, remote)
	}
	return append(argv, remote, transfer.Local)
}

func cpStatusForBox(ref string) (*cpStatusResponse, error) {
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()

	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		return nil, err
	}

	sshStatus, err := rt.GetSshStatus(target.ID, service.LocalUserID)
	if err != nil {
		return nil, err
	}

	status := &cpStatusResponse{Ready: sshStatus.Ready}
	if sshStatus.Instance != nil {
		box := instancesToBoxes([]*service.Instance{sshStatus.Instance})[0]
		status.Instance = &box
	}
	return status, nil
}

// CP copies one file between the local machine and a devbox using scp.
func CP(args []string) {

	fs := flag.NewFlagSet("cp", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: devbox cp [-i key] <source> <dest>")
		fmt.Fprintln(os.Stderr, "examples:")
		fmt.Fprintln(os.Stderr, "  devbox cp ./main.go mybox:/home/ec2-user/app/")
		fmt.Fprintln(os.Stderr, "  devbox cp mybox:/home/ec2-user/app/main.go ./")
	}

	parsed, err := helper.ParseCPCommandArgs(args, cpDefaultKeyPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "cp: %v\n", err)
		fs.Usage()
		os.Exit(1)
	}

	transfer, err := parseCPTransfer(parsed.Source, parsed.Dest) //figure out upload vs download and set the transfer struct
	if err != nil {
		fmt.Fprintf(os.Stderr, "cp: %v\n", err)
		os.Exit(1)
	}

	status, err := cpStatusForBox(transfer.BoxRef) //look up the box status
	if err != nil {
		fmt.Fprintf(os.Stderr, "cp: %v\n", err)
		os.Exit(1)
	}
	if !status.Ready {
		fmt.Fprintln(os.Stderr, "cp: box is not ready yet.")
		os.Exit(1)
	}
	if status.Instance == nil {
		fmt.Fprintln(os.Stderr, "cp: server reported ready but returned no instance details, try the command again in a few minutes.")
		os.Exit(1)
	}
	if status.Instance.PublicIP == "" {
		fmt.Fprintln(os.Stderr, "cp: box has no IP address (is it running?)")
		os.Exit(1)
	}
	if status.Instance.Status != "running" {
		fmt.Fprintf(os.Stderr, "cp: box is %s, not running\n", status.Instance.Status)
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh") //find ssh on machine
	if err != nil {
		fmt.Fprintln(os.Stderr, "cp: ssh binary not found in PATH")
		os.Exit(1)
	}

	scpBin, err := exec.LookPath("scp") //find scp on machine
	if err != nil {
		fmt.Fprintln(os.Stderr, "cp: scp binary not found in PATH")
		os.Exit(1)
	}

	// check if the box is ready
	if err := waitForDevboxReady(sshBin, parsed.Identity, cpDefaultSSHUser, status.Instance.PublicIP, cpDefaultSSHPort); err != nil {
		fmt.Fprintf(os.Stderr, "cp: %v\n", err)
		os.Exit(1)
	}

	// build the scp command
	argv := buildSCPArgs(parsed.Identity, transfer, cpDefaultSSHUser, status.Instance.PublicIP, cpDefaultSSHPort)
	cmd := exec.Command(scpBin, argv...) // run the scp command
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "cp: scp failed: %v\n", err)
		os.Exit(1)
	}
}
