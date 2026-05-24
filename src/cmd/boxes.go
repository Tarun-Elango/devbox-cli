package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devbox-cli/internal/api"
	"devbox-cli/internal/config"
)

// readPublicKey returns the contents of the user's default SSH public key,
// trying id_ed25519.pub then id_rsa.pub under ~/.ssh.
func readPublicKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	for _, name := range []string{"id_ed25519.pub", "id_rsa.pub"} {
		p := filepath.Join(home, ".ssh", name)
		data, err := os.ReadFile(p)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}
	return "", fmt.Errorf("no public key found in ~/.ssh (tried id_ed25519.pub, id_rsa.pub)")
}

// Box represents a devbox instance as returned by the API.
type Box struct {
	ID               string `json:"instanceId"`
	Name             string `json:"name"`
	Status           string `json:"state"`
	InstanceType     string `json:"instanceType"`
	PublicIP         string `json:"publicIpAddress"`
	PrivateIP        string `json:"privateIpAddress"`
}

// Ls lists all boxes belonging to the authenticated user.
func Ls() {
	if TestMode {
		fmt.Println("[test] ls: done")
		return
	}
	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Get("/v1/boxes")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ls failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "ls failed: %v\n", err)
		os.Exit(1)
	}

	var boxes []Box
	if err := api.DecodeJSON(resp, &boxes); err != nil {
		fmt.Fprintf(os.Stderr, "ls failed: %v\n", err)
		os.Exit(1)
	}

	if len(boxes) == 0 {
		fmt.Println("No boxes found.")
		return
	}

	fmt.Printf("%-24s  %-20s  %-10s  %-16s\n", "ID", "NAME", "STATUS", "PUBLIC IP")
	fmt.Println(strings.Repeat("-", 80))
	for _, b := range boxes {
		fmt.Printf("%-24s  %-20s  %-10s  %-16s\n", b.ID, b.Name, b.Status, b.PublicIP)
	}
}

// Status displays details for a single box.
func Status(args []string) {
	if TestMode {
		fmt.Println("[test] status: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox status <id>")
		os.Exit(1)
	}
	id := args[0]

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Get("/v1/boxes/" + id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
		os.Exit(1)
	}

	var b Box
	if err := api.DecodeJSON(resp, &b); err != nil {
		fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ID:        %s\n", b.ID)
	fmt.Printf("Name:      %s\n", b.Name)
	fmt.Printf("Status:    %s\n", b.Status)
	fmt.Printf("Public IP:  %s\n", b.PublicIP)
	fmt.Printf("Private IP: %s\n", b.PrivateIP)
	fmt.Printf("Type:       %s\n", b.InstanceType)
}

// Create creates a new box with an optional name and streams progress via WebSocket.
// Pass --from <snapshot_ami_id> to restore from a previously saved snapshot.
func Create(args []string) {
	if TestMode {
		fmt.Println("[test] create: done")
		return
	}

	// Parse positional name and optional --from <snapshot_ami_id> flag.
	var name, fromSnapshot string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --from requires a snapshot AMI ID")
				os.Exit(1)
			}
			i++
			fromSnapshot = args[i]
		default:
			if name == "" {
				name = strings.TrimSpace(args[i])
			}
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: devbox create <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}

	body := map[string]string{"name": name}

	if fromSnapshot != "" {
		body["fromSnapshot"] = fromSnapshot
	}

	// Include the user's public key if available, so they can SSH in without extra setup.
	if pubKey, err := readPublicKey(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; box will be created without your public key\n", err)
	} else {
		body["publicKey"] = pubKey
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Use a long timeout: the server blocks until EC2 status checks pass (up to ~10 min).
	client := api.NewWithTimeout(cfg.ServerURL, cfg.Token, 15*time.Minute)

	if fromSnapshot != "" {
		fmt.Printf("Restoring box %q from snapshot AMI %s - waiting for it to be ready (this may take a few minutes)...\n", name, fromSnapshot)
	} else {
		fmt.Printf("Creating box %q — waiting for it to be ready (this may take a few minutes)...\n", name)
	}

	resp, err := client.Post("/v1/boxes", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "create failed: %v\n", err)
		os.Exit(1)
	}

	var b Box
	if err := api.DecodeJSON(resp, &b); err != nil {
		fmt.Fprintf(os.Stderr, "create failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Box is ready.\n")
	fmt.Printf("  ID:        %s\n", b.ID)
	fmt.Printf("  Name:      %s\n", b.Name)
	if b.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", b.PublicIP)
		fmt.Printf("\n  Connect:   devbox ssh %s\n", b.ID)
	}
}

// httpToWS converts an http(s) base URL to its ws(s) equivalent.
func httpToWS(serverURL string) string {
	switch {
	case strings.HasPrefix(serverURL, "https://"):
		return "wss://" + serverURL[len("https://"):]
	case strings.HasPrefix(serverURL, "http://"):
		return "ws://" + serverURL[len("http://"):]
	default:
		return serverURL
	}
}

// Stop stops a running box.
func Stop(args []string) {
	if TestMode {
		fmt.Println("[test] stop: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox stop <id>")
		os.Exit(1)
	}
	id := args[0]

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Post("/v1/boxes/"+id+"/stop", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stop failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "stop failed: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()

	fmt.Printf("Box %s stopped.\n", id)
}

// Start starts a stopped box.
func Start(args []string) {
	if TestMode {
		fmt.Println("[test] start: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox start <id>")
		os.Exit(1)
	}
	id := args[0]

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Post("/v1/boxes/"+id+"/start", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "start failed: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()

	fmt.Printf("Box %s started.\n", id)
}

// Delete permanently deletes a box.
func Delete(args []string) {
	if TestMode {
		fmt.Println("[test] delete: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox delete <id>")
		os.Exit(1)
	}
	id := args[0]

	fmt.Printf("Are you sure you want to delete box %s? [y/N] ", id)
	var answer string
	fmt.Scanln(&answer)
	if answer != "y" {
		fmt.Println("Aborted.")
		return
	}

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Delete("/v1/boxes/" + id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "delete failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "delete failed: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()

	fmt.Printf("Box %s deleted.\n", id)
}
