package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/helper"
	"devbox-cli/service"
)

const templateNewUsageLine = "usage: devbox template new <name> [command string]"

// FailBox prints a clean error for a box subcommand and exits.
func FailBox(cmd string, err error) {
	fmt.Fprintf(os.Stderr, "%s failed: %s\n", cmd, err)
	os.Exit(1)
}

// Template dispatches template sub-commands.
//
//	devbox template                                → list available templates
//	devbox template new <name> [command string]    → create a template
//	devbox template delete <name>                    → delete a template
//	devbox template rename <name> <new-name>         → rename a template
//	devbox template search <query>                   → search templates by name
func Template(args []string) {
	if len(args) == 0 {
		TemplateList(args)
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "new":
		TemplateNew(subArgs)
	case "delete":
		TemplateDelete(subArgs)
	case "rename":
		TemplateRename(subArgs)
	case "search":
		TemplateSearch(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown subcommand %q (expected %q, %q, %q, or %q)\n", sub, "new", "delete", "rename", "search")
		os.Exit(1)
	}
}

// ParseTemplateNewArgs parses arguments after the "template new" subcommand.
// Expected shape: <name> [command parts...]
func ParseTemplateNewArgs(args []string) (name, startupScript string, err error) {
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
		startupScript = strings.Join(args[1:], " ") // join the args with a space
		startupScript = strings.TrimSpace(startupScript)
	}
	return name, startupScript, nil
}

func templateNewUsage() string {
	return templateNewUsageLine
}

// TemplateNew creates a user-owned startup template.
// Usage: devbox template new <name> [command string]

// this is to create a new template
func TemplateNew(args []string) {
	name, startupScript, err := ParseTemplateNewArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, templateNewUsage())
		os.Exit(1)
	}

	var created service.Template
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	tmpl, err := rt.CreateTemplate(name, startupScript, service.LocalUserID)
	if err != nil {
		FailBox("template new", err)
	}
	created = *tmpl

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

const templateDeleteUsageLine = "usage: devbox template delete <name>"

// TemplateDelete deletes a user-owned startup template.
// Usage: devbox template delete <name>
func TemplateDelete(args []string) {
	id := strings.TrimSpace(helper.ParseTemplateDeleteArgs(args, templateDeleteUsageLine))
	if id == "" {
		fmt.Fprintln(os.Stderr, "error: template name is required")
		fmt.Fprintln(os.Stderr, templateDeleteUsageLine)
		os.Exit(1)
	}
	if strings.HasPrefix(id, "--") {
		fmt.Fprintf(os.Stderr, "error: unknown flag %q\n", id)
		os.Exit(1)
	}

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	if err := rt.DeleteTemplate(id, service.LocalUserID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "template %s not found\n", id)
		} else {
			fmt.Fprintf(os.Stderr, "template delete failed: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("Template %s deleted.\n", id)
}

const templateRenameUsageLine = "usage: devbox template rename <name> <new-name>"

// TemplateRename updates a user-owned template name.
// Usage: devbox template rename <name> <new-name>
func TemplateRename(args []string) {
	id, newName := helper.ParseTemplateRenameArgs(args, templateRenameUsageLine)
	id = strings.TrimSpace(id)
	newName = strings.TrimSpace(newName)
	if id == "" {
		fmt.Fprintln(os.Stderr, "error: template name is required")
		fmt.Fprintln(os.Stderr, templateRenameUsageLine)
		os.Exit(1)
	}
	if newName == "" {
		fmt.Fprintln(os.Stderr, "error: new template name is required")
		fmt.Fprintln(os.Stderr, templateRenameUsageLine)
		os.Exit(1)
	}
	if strings.HasPrefix(id, "--") || strings.HasPrefix(newName, "--") {
		fmt.Fprintf(os.Stderr, "error: unknown flag\n")
		os.Exit(1)
	}

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	renamed, err := rt.RenameTemplate(id, newName, service.LocalUserID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "template %s not found\n", id)
		} else if strings.Contains(err.Error(), "already exists") {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "template rename failed: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("Template %s renamed to %s.\n", id, renamed.Name)
}
