package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"devbox-cli/internal/api"
	"devbox-cli/internal/config"
)
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StartupScript string `json:"startupScript"`
}

func Templates(args []string) {
	fmt.Println("Fetching Templates")
	if TestMode {
		fmt.Println("[test] templates: done")
		return
	}

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Get("/v1/boxes/templates")
	if err != nil {
		api.FailBox("templates", err)
	}
	
	if err := api.CheckStatus(resp); err != nil {
		api.FailBox("templates", err)
	}

	// build a table of templates
	var templates []Template
	if err := api.DecodeJSON(resp, &templates); err != nil { // decode the response body into the templates slice
		api.FailBox("templates", err)
	}

	fmt.Printf("%-24s  %-20s  %-10s\n", "ID", "NAME", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 100))
	for _, t := range templates {
		fmt.Printf("%-24s  %-20s  %-10s\n", t.ID, t.Name, t.Description)
	}
}
// notes: check valid template id, name cannot start with --, 
// -- from should be valid string and should have a snapshot ami id
func CreateTemplate(args []string) {

	// -args wont have --template flag
	if TestMode {
		fmt.Println("[test] create template: done")
		return
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox create --template <templateId> [<templateId>...] <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}

	var positionalArgs []string
	var name, fromSnapshot string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --from requires a snapshot AMI ID")
				os.Exit(1)
			}
			i++
			if err := validateSnapshotAmiID(args[i]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fromSnapshot = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(os.Stderr, "error: %v\n", unknownCreateFlagError(args[i]))
				os.Exit(1)
			}
			arg := strings.TrimSpace(args[i])
			if arg == "" {
				fmt.Fprintln(os.Stderr, "error: template ID is required")
				os.Exit(1)
			}
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox create --template <templateId> [<templateId>...] <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}

	name = positionalArgs[len(positionalArgs)-1]
	templateIDs := positionalArgs[:len(positionalArgs)-1]

	if len(templateIDs) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one template ID is required")
		os.Exit(1)
	}
	for _, id := range templateIDs {
		if strings.HasPrefix(id, "--") {
			fmt.Fprintln(os.Stderr, "error: template ID is required")
			os.Exit(1)
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: devbox create --template <templateId> [<templateId>...] <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}
	if strings.HasPrefix(name, "--") {
		fmt.Fprintln(os.Stderr, "error: box name cannot start with --")
		os.Exit(1)
	}

	body := map[string]any{"name": name, "templateIds": templateIDs}
	if fromSnapshot != "" {
		body["fromSnapshot"] = fromSnapshot
	}
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

	client := api.NewWithTimeout(cfg.ServerURL, cfg.Token, 15*time.Minute)

	if fromSnapshot != "" {
		fmt.Printf("Creating box %q from templates %s (snapshot %s) — waiting for it to be ready (this may take a few minutes)...\n", name, strings.Join(templateIDs, ", "), fromSnapshot)
	} else {
		fmt.Printf("Creating box %q from templates %s — waiting for it to be ready (this may take a few minutes)...\n", name, strings.Join(templateIDs, ", "))
	}

	resp, err := client.Post("/v1/boxes/templates", body)
	if err != nil {
		api.FailBox("create template", err)
	}
	if err := api.CheckStatus(resp); err != nil {
		api.FailBox("create template", err)
	}

	var b Box
	if err := api.DecodeJSON(resp, &b); err != nil {
		api.FailBox("create template", err)
	}

	fmt.Printf("Box is ready.\n")
	fmt.Printf("  ID:        %s\n", b.ID)
	fmt.Printf("  Name:      %s\n", b.Name)
	if b.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", b.PublicIP)
		fmt.Printf("\n  Connect:   devbox ssh %s\n", b.ID)
	}
}