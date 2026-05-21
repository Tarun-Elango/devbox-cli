package cmd

import (
	"fmt"
	"os"

	"devbox-cli/internal/api"
)

// Snapshot creates a snapshot of the given box.
// Usage: devbox snapshot <id>
func Snapshot(args []string) {
	if TestMode {
		fmt.Println("[test] snapshot: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox snapshot <id>")
		os.Exit(1)
	}
	id := args[0]

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Post("/v1/boxes/"+id+"/snapshots", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}

	var result struct {
		ID        string `json:"id"`
		CreatedAt string `json:"createdAt"`
	}
	if err := api.DecodeJSON(resp, &result); err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}

	if result.ID != "" {
		fmt.Printf("Snapshot created: %s", result.ID)
		if result.CreatedAt != "" {
			fmt.Printf(" (created %s)", result.CreatedAt)
		}
		fmt.Println()
	} else {
		fmt.Printf("Snapshot created for box %s.\n", id)
	}
}
