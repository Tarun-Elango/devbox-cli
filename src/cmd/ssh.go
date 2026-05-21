package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"devbox-cli/internal/api"
)

// SSH fetches the box's IP address and execs ssh, replacing the current process.
func SSH(args []string) {
	if TestMode {
		fmt.Println("[test] ssh: done")
		return
	}
	fs := flag.NewFlagSet("ssh", flag.ExitOnError)
	user := fs.String("u", "root", "SSH username")
	port := fs.Int("p", 22, "SSH port")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: devbox ssh [-u user] [-p port] <id> [-- ssh-args...]")
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

	if b.IP == "" {
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

	target := fmt.Sprintf("%s@%s", *user, b.IP)
	portArg := fmt.Sprintf("%d", *port)

	argv := []string{sshBin, "-p", portArg, target}
	argv = append(argv, extra...)

	if err := syscall.Exec(sshBin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "ssh: exec failed: %v\n", err)
		os.Exit(1)
	}
}
