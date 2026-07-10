package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"outpost-cli/helper"
	"outpost-cli/service"
)

// Import syncs untracked EC2 instances and self-owned AMIs from the configured
// AWS region into the local DB, prompting yes/no for each. For boxes, optionally
// prompts for an existing private key (.pem) and appends the local outpost
// public key to authorized_keys so outpost ssh works.
func Import(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost import")

	fmt.Fprintln(os.Stderr, "warning: if an instance is not Amazon Linux, commands like outpost idle-stop and outpost ssh will not work")

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()

	candidates, err := rt.ListUntrackedImportCandidates(service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(candidates) == 0 {
		fmt.Println("Nothing new to import — all instances and snapshots in this region are already tracked.")
		return
	}

	var imported int
	for _, c := range candidates {
		kind := "box"
		extra := c.State
		if c.Kind == service.ImportKindSnapshot {
			kind = "snapshot"
		} else if c.InstanceType != "" {
			extra = c.State + ", " + c.InstanceType
		}
		fmt.Printf("Import %s %s (%s) as %q? [y/N] ", kind, c.AWSID, extra, c.Name)

		line, err := helper.ReadStdinLine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if line != "y" && line != "Y" {
			fmt.Printf("  skipped %s\n", c.AWSID)
			continue
		}

		identityPath := ""
		offerAuthorize := false
		if c.Kind == service.ImportKindBox { // If the candidate is a box, offer to authorize SSH access
			if c.IPAddress != "" && strings.EqualFold(c.State, "running") {
				// If the box has a reachable IP and is running, offer to authorize SSH access
				offerAuthorize = true
				identityPath = promptImportIdentityPath()
			} else {
				fmt.Println("  (box has no reachable IP or is not running — skipping SSH authorize)")
			}
		}

		if err := rt.ImportCandidate(c, service.LocalUserID); err != nil {
			fmt.Fprintf(os.Stderr, "  error importing %s: %v\n", c.AWSID, err)
			continue
		}
		imported++
		fmt.Printf("  imported %s as %q\n", c.AWSID, c.Name)

		if c.Kind != service.ImportKindBox {
			continue
		}
		if c.IPAddress != "" {
			addSSHHostOrWarn(c.Name, &service.Instance{Name: c.Name, IPAddress: c.IPAddress, Status: c.State})
		}
		switch {
		case identityPath != "": // If the user provided a private key path, authorize SSH access
			if err := authorizeImportedBox(identityPath, c.IPAddress); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: could not authorize outpost SSH key: %v\n", err)
				printImportAuthorizeManual(c, identityPath)
			} else {
				fmt.Printf("  authorized outpost SSH key on %s — try: outpost ssh %s\n", c.Name, c.Name)
			}
		case offerAuthorize:
			printImportAuthorizeManual(c, "")
		}
	}

	fmt.Printf("Done. Imported %d of %d.\n", imported, len(candidates))
}

// promptImportIdentityPath asks for an existing private key path used to reach
// the box once. Blank skips. Re-prompts when the path does not exist.
func promptImportIdentityPath() string {
	for {
		fmt.Print("  Path to existing SSH private key (.pem) so outpost can log in (example: ~/Downloads/my-key.pem; leave blank to skip): ")
		line, err := helper.ReadStdinLine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: could not read key path: %v\n", err)
			return ""
		}
		path, err := expandUserPath(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: %v\n", err)
			continue
		}
		if path == "" {
			return ""
		}
		if st, err := os.Stat(path); err != nil || st.IsDir() {
			fmt.Fprintf(os.Stderr, "  file not found: %s (try again, or leave blank to skip)\n", path)
			continue
		}
		return path
	}
}

func printImportAuthorizeManual(c service.ImportCandidate, identityPath string) {
	if c.IPAddress == "" {
		fmt.Println("  note: outpost ssh needs ~/.ssh/id_ed25519.pub in the box's authorized_keys (import does not inject keys)")
		return
	}
	keyFlag := "-i /path/to/key.pem"
	if identityPath != "" {
		keyFlag = "-i " + identityPath
	}
	fmt.Printf("  note: add your outpost public key once, then use outpost ssh:\n")
	fmt.Printf("    ssh %s ec2-user@%s 'mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys' < ~/.ssh/id_ed25519.pub\n",
		keyFlag, c.IPAddress)
}

// expandUserPath strips quotes and expands a leading ~/ to the home directory.
func expandUserPath(raw string) (string, error) {
	p := helper.StripSurroundingQuotes(strings.TrimSpace(raw))
	if p == "" {
		return "", nil
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if p == "~" {
			return home, nil
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

// buildAuthorizeRemoteCommand returns a remote shell snippet that appends
// publicKey to ~/.ssh/authorized_keys if it is not already present. It also
// best-effort creates the outpost ready marker: imported boxes have no outpost
// user-data or templates to wait for, and outpost ssh uses this marker.
func buildAuthorizeRemoteCommand(publicKey string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(publicKey)))
	return fmt.Sprintf(
		`KEY=$(echo '%s' | base64 -d) && mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && { grep -qxF "$KEY" ~/.ssh/authorized_keys || echo "$KEY" >> ~/.ssh/authorized_keys; } && { { sudo -n mkdir -p /var/lib/outpost && printf '%%s\n' %q | sudo -n tee %s >/dev/null; } || true; }`,
		encoded, outpostReadyMessage, outpostReadyPath,
	)
}

// authorizeImportedBox SSHes with identity and appends the local outpost
// public key to the box's authorized_keys. Does not wait for outpost ready
// markers (imported boxes typically have none).
func authorizeImportedBox(identity, host string) error {
	pubKey, err := readPublicKey()
	if err != nil {
		return err
	}
	if strings.TrimSpace(pubKey) == "" {
		return fmt.Errorf("empty public key")
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh binary not found in PATH")
	}

	argv := sshBaseArgs(identity, defaultSSHPort)
	argv = append(argv,
		"-o", "BatchMode=yes",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", defaultSSHUser, host),
		buildAuthorizeRemoteCommand(pubKey),
	)

	cmd := execCommand(sshBin, argv...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh with %s failed: %w", identity, err)
	}
	return nil
}
