package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"devbox-cli/internal/api"
	"devbox-cli/internal/config"
	"devbox-cli/internal/ws"
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
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	IP        string `json:"ip"`
	Template  string `json:"template"`
	CreatedAt string `json:"createdAt"`
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

	fmt.Printf("%-24s  %-20s  %-10s  %-16s  %s\n", "ID", "NAME", "STATUS", "IP", "CREATED")
	fmt.Println(strings.Repeat("-", 90))
	for _, b := range boxes {
		fmt.Printf("%-24s  %-20s  %-10s  %-16s  %s\n", b.ID, b.Name, b.Status, b.IP, b.CreatedAt)
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
	fmt.Printf("IP:        %s\n", b.IP)
	fmt.Printf("Template:  %s\n", b.Template)
	fmt.Printf("Created:   %s\n", b.CreatedAt)
}

// Create creates a new box with an optional name and streams progress via WebSocket.
func Create(args []string) {
	if TestMode {
		fmt.Println("[test] create: done")
		return
	}
	body := map[string]string{}
	if len(args) >= 1 {
		body["name"] = args[0]
	}

	// Include the user's public key if available, so they can SSH in without extra setup.
	if pubKey, err := readPublicKey(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; box will be created without your public key\n", err)
	} else {
		body["public_key"] = pubKey
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	client := api.New(cfg.ServerURL, cfg.Token)

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

	fmt.Printf("Created box %s (%s). Waiting for it to start...\n", b.Name, b.ID)

	// Watch the box's status via WebSocket until it reaches a terminal state.
	if err := watchBox(cfg.ServerURL, cfg.Token, b.ID); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not watch box status: %v\n", err)
		fmt.Printf("Box ID: %s\n", b.ID)
		return
	}
}

// watchBox connects to the WebSocket watch endpoint for boxID and prints
// status events until the box reaches a terminal state or the server closes
// the connection.
func watchBox(serverURL, token, boxID string) error {
	wsURL := httpToWS(serverURL) + "/v1/boxes/" + boxID + "/watch"

	conn, err := ws.Dial(wsURL, token)
	if err != nil {
		return fmt.Errorf("dial watch endpoint: %w", err)
	}
	defer conn.Close()

	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read event: %w", err)
		}

		var event struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}
		if jsonErr := json.Unmarshal([]byte(msg), &event); jsonErr != nil {
			// Non-JSON message — print raw.
			fmt.Println(msg)
			continue
		}

		if event.Message != "" {
			fmt.Printf("  %s\n", event.Message)
		} else if event.Status != "" {
			fmt.Printf("  status: %s\n", event.Status)
		}

		switch event.Status {
		case "running":
			fmt.Println("Box is running.")
			return nil
		case "error", "failed":
			return fmt.Errorf("box reached error state: %s", event.Status)
		}
	}

	fmt.Println("Box provisioning complete.")
	return nil
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
	resp.Body.Close()
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "stop failed: %v\n", err)
		os.Exit(1)
	}

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
	resp.Body.Close()
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "start failed: %v\n", err)
		os.Exit(1)
	}

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
	resp.Body.Close()
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "delete failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Box %s deleted.\n", id)
}
