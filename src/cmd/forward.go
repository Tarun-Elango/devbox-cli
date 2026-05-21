package cmd

import (
	"fmt"
	"os"

	"devbox-cli/internal/api"
)

// Forward requests a port-forward for a box and prints the forwarded URL.
// Usage: devbox forward <id> <port>
func Forward(args []string) {
	if TestMode {
		fmt.Println("[test] forward: done")
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox forward <id> <port>")
		os.Exit(1)
	}
	id := args[0]
	port := args[1]

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	body := map[string]string{"port": port}
	resp, err := client.Post("/v1/boxes/"+id+"/ports", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
		os.Exit(1)
	}

	var result struct {
		URL  string `json:"url"`
		Port string `json:"port"`
	}
	if err := api.DecodeJSON(resp, &result); err != nil {
		fmt.Fprintf(os.Stderr, "forward failed: %v\n", err)
		os.Exit(1)
	}

	if result.URL != "" {
		fmt.Printf("Forwarded port %s → %s\n", port, result.URL)
	} else {
		fmt.Printf("Forwarded port %s on box %s.\n", port, id)
	}
}
