package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"devbox-cli/internal/api"
)

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

// SSH fetches the box's IP address and execs ssh, replacing the current process.
func SSH(args []string) {
	if TestMode {
		fmt.Println("[test] ssh: done")
		return
	}
	fs := flag.NewFlagSet("ssh", flag.ExitOnError)
	user := fs.String("u", "ec2-user", "SSH username")
	port := fs.Int("p", 22, "SSH port")
	identity := fs.String("i", defaultKeyPath(), "path to SSH private key")
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

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Get("/v1/boxes/" + id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(1)
	}

	var b Box
	if err := api.DecodeJSON(resp, &b); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(1)
	}

	if b.PublicIP == "" {
		fmt.Fprintln(os.Stderr, "ssh: box has no IP address (is it running?)")
		os.Exit(1)
	}
	if b.Status != "running" {
		fmt.Fprintf(os.Stderr, "ssh: box is %s, not running\n", b.Status)
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh") // LookPath only returns an error if the binary isn't found, so we don't need to check for exec.ErrNotFound specifically.
	if err != nil {
		fmt.Fprintln(os.Stderr, "ssh: ssh binary not found in PATH")
		os.Exit(1)
	}

	target := fmt.Sprintf("%s@%s", *user, b.PublicIP)
	portArg := fmt.Sprintf("%d", *port)

	fmt.Fprintf(os.Stderr, "Connecting to %s (box %s)...\n", target, id)

	argv := []string{sshBin,
		"-p", portArg,
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	// identity is optional, so only include it if the user specified one (either via -i or defaultKeyPath).
	if *identity != "" {
		argv = append(argv, "-i", *identity)
	}
	argv = append(argv, target)
	argv = append(argv, extra...)

	if err := syscall.Exec(sshBin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: exec failed: %v\n", err)
		os.Exit(1)
	}
}
