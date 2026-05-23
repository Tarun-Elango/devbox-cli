package cmd

import (
	"fmt"
	"os"

	"devbox-cli/internal/api"
)

// Snapshot creates an AMI snapshot of the given box.
// Usage: devbox snapshot <id> [name]
func Snapshot(args []string) {
	if TestMode {
		fmt.Println("[test] snapshot: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox snapshot <id> [name]")
		os.Exit(1)
	}
	id := args[0]
	name := "snapshot-" + id
	if len(args) >= 2 {
		name = args[1]
	}

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	body := map[string]string{"name": name}
	resp, err := client.Post("/v1/boxes/"+id+"/snapshots", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}

	var result struct {
		AmiID string `json:"amiId"`
		Name  string `json:"name"`
		State string `json:"state"`
	}
	if err := api.DecodeJSON(resp, &result); err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Snapshot created: %s  name=%s  state=%s\n", result.AmiID, result.Name, result.State)
}
