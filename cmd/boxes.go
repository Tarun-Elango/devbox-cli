package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"devbox-cli/service"
)

// mostly just util and helper functions for the boxes command

// readPublicKey returns the contents of ~/.ssh/id_ed25519.pub.
// If both key files exist, it returns the public key. If neither exists, it
// creates a new pair without prompting. If only one exists, it prompts before
// repairing/creating keys via ensureEd25519Key (private-only → derive pub;
// public-only → replace with a new pair).
func readPublicKey() (string, error) {
	priv, pub, err := ed25519KeyPaths()
	if err != nil {
		return "", err
	}

	_, privErr := os.Stat(priv)
	_, pubErr := os.Stat(pub)
	privExists := privErr == nil
	pubExists := pubErr == nil

	if privExists && pubExists {
		data, err := os.ReadFile(pub)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", pub, err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	// Neither key exists: just create both, no need to prompt.
	if !privExists && !pubExists {
		if err := ensureEd25519Key(); err != nil {
			return "", err
		}

		fmt.Printf("SSH key pair created in your ~/.ssh directory.\n")

		data, err := os.ReadFile(pub)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", pub, err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	// One key exists without the other: confirm before repairing/replacing.
	if privExists && !pubExists {
		fmt.Printf("Private key found at %s but public key is missing. Derive the public key from it? [y/N] ", priv)
	} else {
		fmt.Printf("SSH public key found at %s but private key is missing. Generate a new key pair (replaces the public key)? [y/N] ", pub)
	}
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "y" && answer != "Y" {
		if !pubExists {
			return "", fmt.Errorf("no public key at %s", pub)
		}
		return "", fmt.Errorf("private key missing at %s", priv)
	}

	if err := ensureEd25519Key(); err != nil {
		return "", err
	}

	fmt.Printf("SSH public key created in your ~/.ssh directory.\n")

	data, err := os.ReadFile(pub)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", pub, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ensureEd25519Key makes sure ~/.ssh/id_ed25519 and .pub both exist:
// private-only → derive the public key; public-only → replace with a new pair;
// neither → create both. Never overwrites an existing private key.
func ensureEd25519Key() error {
	priv, pub, err := ed25519KeyPaths()
	if err != nil {
		return err
	}

	_, privErr := os.Stat(priv)
	_, pubErr := os.Stat(pub)
	privExists := privErr == nil
	pubExists := pubErr == nil

	if privExists && pubExists {
		return nil
	}

	sshKeygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return fmt.Errorf("ssh-keygen not found in PATH")
	}

	if err := os.MkdirAll(filepath.Dir(priv), 0o700); err != nil {
		return fmt.Errorf("create ~/.ssh: %w", err)
	}

	// Private key present, public missing: derive .pub from the existing private key.
	if privExists && !pubExists {
		cmd := exec.Command(sshKeygen, "-y", "-f", priv)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("derive public key from %s: %w", priv, err)
		}
		line := strings.TrimSpace(string(out)) + "\n"
		if err := os.WriteFile(pub, []byte(line), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", pub, err)
		}
		return nil
	}

	// Public-only: orphaned .pub cannot authenticate; replace with a new key pair.
	if pubExists && !privExists {
		if err := os.Remove(pub); err != nil {
			return fmt.Errorf("remove orphaned public key %s: %w", pub, err)
		}
	}

	cmd := exec.Command(sshKeygen, "-t", "ed25519", "-f", priv)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen failed: %w", err)
	}
	return nil
}

// Box represents a devbox instance as returned by the API.
type Box struct {
	ID           string `json:"instanceId"`
	Name         string `json:"name"`
	Status       string `json:"state"`
	InstanceType string `json:"instanceType"`
	PublicIP     string `json:"publicIpAddress"`
	PrivateIP    string `json:"privateIpAddress"`
	Region       string `json:"region"`
	Provider     string `json:"provider"`
}

func instancesToBoxes(instances []*service.Instance) []Box {
	boxes := make([]Box, len(instances))
	for i, inst := range instances {
		boxes[i] = Box{
			ID:           inst.ID,
			Name:         inst.Name,
			Status:       inst.Status,
			InstanceType: inst.InstanceType,
			PublicIP:     inst.IPAddress,
			PrivateIP:    inst.PrivateIPAddress,
			Region:       inst.Region,
			Provider:     inst.Provider,
		}
	}
	return boxes
}

// addSSHHostOrWarn writes the box into ~/.ssh/config; failures are non-fatal.
func addSSHHostOrWarn(name string, inst *service.Instance) {
	ip, err := inst.SSHHost()
	if err != nil {
		return
	}
	if err := service.AddHost(name, ip); err != nil {
		fmt.Fprintf(os.Stderr, "warning: box created but failed to update SSH config on this machine (~/.ssh/config): %v\n", err)
		return
	}
	fmt.Printf("  SSH config: devbox-%s added to ~/.ssh/config\n", name)
}
