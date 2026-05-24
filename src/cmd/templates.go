package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/internal/api"
)

// Template represents a devbox template as returned by the API.
type Template struct {
	AmiID       string `json:"amiId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
}

// Templates lists all available box templates.
func Templates() {
	if TestMode {
		fmt.Println("[test] templates: done")
		return
	}
	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Get("/v1/templates")
	if err != nil {
		fmt.Fprintf(os.Stderr, "templates failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "templates failed: %v\n", err)
		os.Exit(1)
	}

	var templates []Template
	if err := api.DecodeJSON(resp, &templates); err != nil {
		fmt.Fprintf(os.Stderr, "templates failed: %v\n", err)
		os.Exit(1)
	}

	if len(templates) == 0 {
		fmt.Println("No templates available.")
		return
	}

	fmt.Printf("%-24s  %-20s  %-12s  %s\n", "AMI ID", "NAME", "STATE", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 90))
	for _, t := range templates {
		fmt.Printf("%-24s  %-20s  %-12s  %s\n", t.AmiID, t.Name, t.State, t.Description)
	}
}
