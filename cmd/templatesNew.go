package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/internal/api"
)

const templateNewUsageLine = "usage: devbox template new <name> [command string]"

// ParseTemplateNewArgs parses arguments after the top-level "template" command.
// Expected shape: new <name> [command parts...]
func ParseTemplateNewArgs(args []string) (name, startupScript string, err error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("missing subcommand %q", "new")
	}
	if args[0] != "new" {
		return "", "", fmt.Errorf("unknown subcommand %q (expected %q)", args[0], "new")
	}
	args = args[1:]
	if len(args) == 0 {
		return "", "", fmt.Errorf("template name is required")
	}

	name = strings.TrimSpace(args[0])
	if name == "" {
		return "", "", fmt.Errorf("template name is required")
	}
	if strings.HasPrefix(name, "--") {
		return "", "", fmt.Errorf("template name cannot be a flag")
	}

	// remaining args is the startup script
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--") {
			return "", "", fmt.Errorf("unknown flag %q", arg)
		}
	}

	if len(args) > 1 {
		startupScript = strings.Join(args[1:], " ")// join the args with a space
		startupScript = strings.TrimSpace(startupScript)
	}
	return name, startupScript, nil
}

func templateNewUsage() string {
	return templateNewUsageLine
}

// TemplateNew creates a user-owned startup template.
// Usage: devbox template new <name> [command string]
func TemplateNew(args []string) {
	name, startupScript, err := ParseTemplateNewArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, templateNewUsage())
		os.Exit(1)
	}

	if TestMode {
		fmt.Printf("[test] template new: name=%q", name)
		if startupScript != "" {
			fmt.Printf(" startupScript=%q", startupScript)
		}
		fmt.Println()
		return
	}

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	body := map[string]string{"name": name}
	if startupScript != "" {
		body["startupScript"] = startupScript
	}

	resp, err := client.Post("/v1/boxes/templates", body)
	if err != nil {
		api.FailBox("template new", err)
	}
	if err := api.CheckStatus(resp); err != nil {
		api.FailBox("template new", err)
	}

	var created Template
	if err := api.DecodeJSON(resp, &created); err != nil {
		api.FailBox("template new", err)
	}

	fmt.Printf("Template created.\n")
	fmt.Printf("  Name: %s\n", created.Name)
	if created.Description != "" {
		fmt.Printf("  Description: %s\n", created.Description)
	}
	if startupScript != "" {
		fmt.Printf("\n  Use: devbox create --template %s <box-name>\n", created.Name)
	} else {
		fmt.Printf("\n  Add a startup command later or use as-is with:\n")
		fmt.Printf("  devbox create --template %s <box-name>\n", created.Name)
	}
}
