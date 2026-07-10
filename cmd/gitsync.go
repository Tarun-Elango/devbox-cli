package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"outpost-cli/helper"
	"outpost-cli/service"
)

// GitSync toggles GitHub SSH access for a box. Synced means both the local SSH
// key is loaded in ssh-agent and ForwardAgent yes is set in the box's SSH
// config entry. If either side is missing, git-sync enables both; when both
// are already set, it disables both.
func GitSync(args []string) {
	ref := helper.ParseSingleBoxRef(args, "usage: outpost git-sync <id|name>")

	keyPath := defaultKeyPath()
	if keyPath == "" {
		fmt.Fprintln(os.Stderr, "git-sync: no SSH key found at ~/.ssh/id_ed25519")
		os.Exit(1)
	}

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-sync: %v\n", err)
		os.Exit(1)
	}

	configEnabled, err := service.ForwardAgentEnabled(target.Name) // yes or no
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-sync: %v\n", err)
		os.Exit(1)
	}

	keyLoaded, err := keyInSSHAgent(keyPath) // yes or no
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-sync: %v\n", err)
		os.Exit(1)
	}

	synced := configEnabled && keyLoaded
	hostAlias := service.OutpostHostName(target.Name)

	if synced { // if both are true, disable both
		if err := service.DisableForwardAgent(target.Name); err != nil {
			fmt.Fprintf(os.Stderr, "git-sync: %v\n", err)
			os.Exit(1)
		}
		if err := sshAgentRemove(keyPath); err != nil {
			fmt.Fprintf(os.Stderr, "git-sync: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("git-sync: disabled GitHub SSH for %s\n", hostAlias)
		fmt.Println("git-sync:   removed key from ssh-agent and ForwardAgent from ~/.ssh/config")
		return
	}

	if err := service.EnableForwardAgent(target.Name); err != nil {
		fmt.Fprintf(os.Stderr, "git-sync: %v\n", err)
		os.Exit(1)
	}
	if err := sshAgentAdd(keyPath); err != nil {
		fmt.Fprintf(os.Stderr, "git-sync: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("git-sync: enabled GitHub SSH for %s (key in ssh-agent, ForwardAgent yes in ~/.ssh/config)\n", hostAlias)
	fmt.Printf("git-sync: reconnect with `ssh %s` (or `outpost ssh %s -- -A`)\n", hostAlias, target.Name)
	fmt.Println("git-sync: on the box, verify with: ssh -T git@github.com")
}

// helper: keyFingerprint returns the fingerprint of the key at the given path
func keyFingerprint(keyPath string) (string, error) {
	sshKeygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return "", fmt.Errorf("ssh-keygen not found in PATH")
	}

	cmd := execCommand(sshKeygen, "-lf", keyPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen -lf failed: %w", err)
	}

	// Output: "256 SHA256:abc comment (ED25519)"
	fields := strings.Fields(stdout.String())
	if len(fields) < 2 {
		return "", fmt.Errorf("unexpected ssh-keygen output: %q", strings.TrimSpace(stdout.String()))
	}
	return fields[1], nil
}

// helper: keyInSSHAgent checks if the key is in the ssh-agent
func keyInSSHAgent(keyPath string) (bool, error) {
	fingerprint, err := keyFingerprint(keyPath)
	if err != nil {
		return false, err
	}

	sshAdd, err := exec.LookPath("ssh-add")
	if err != nil {
		return false, fmt.Errorf("ssh-add not found in PATH")
	}

	cmd := execCommand(sshAdd, "-l") // list the keys in the ssh-agent
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil { // if the command fails, return false
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			switch exitErr.ExitCode() {
			case 1:
				return false, nil
			case 2:
				return false, fmt.Errorf("ssh-agent is not running")
			}
		}
		return false, fmt.Errorf("ssh-add -l failed: %w", err)
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.Contains(line, fingerprint) {
			return true, nil
		}
	}
	return false, nil
}

// helper: sshAgentAdd adds the key to the ssh-agent
func sshAgentAdd(keyPath string) error {
	sshAdd, err := exec.LookPath("ssh-add")
	if err != nil {
		return fmt.Errorf("ssh-add not found in PATH")
	}
	cmd := execCommand(sshAdd, keyPath) // add the key to the ssh-agent
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-add failed: %w", err)
	}
	return nil
}

// helper: sshAgentRemove removes the key from the ssh-agent
func sshAgentRemove(keyPath string) error {
	loaded, err := keyInSSHAgent(keyPath)
	if err != nil {
		return err
	}
	if !loaded {
		return nil
	}

	sshAdd, err := exec.LookPath("ssh-add")
	if err != nil {
		return fmt.Errorf("ssh-add not found in PATH")
	}
	cmd := execCommand(sshAdd, "-d", keyPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-add -d failed: %w", err)
	}
	return nil
}
