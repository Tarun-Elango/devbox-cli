package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/internal/api"
	"devbox-cli/service"
)

func Templates(args []string) {
	if TestMode {
		fmt.Println("[test] templates: done")
		return
	}

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var templates []*service.Template
	if mode == "local" {
		fmt.Println("Listing local templates")
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		templates, err = rt.ListTemplates(service.LocalUserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Fetching Templates")
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
		if err := api.DecodeJSON(resp, &templates); err != nil {
			api.FailBox("templates", err)
		}
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return
	}

	const colSep = "  |  "
	fmt.Printf("%-20s%s%s\n", "TEMPLATE", colSep, "STARTUP SCRIPT")
	fmt.Println(strings.Repeat("-", 60))
	for _, t := range templates {
		ref := t.ID
		if ref == "" {
			ref = t.Name
		}
		fmt.Printf("%-20s%s%s\n", ref, colSep, formatTemplateScript(t.StartupScript))
	}
}

func formatTemplateScript(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	const maxLen = 72
	if len(s) > maxLen {
		s = s[:maxLen-3] + "..."
	}
	return s
}

// notes: check valid template id, name cannot start with --,
// -- from should be valid string and should have a snapshot ami id

// this is to create a new box with templates
func CreateTemplate(args []string) {

	// -args wont have --template flag
	if TestMode {
		fmt.Println("[test] create template: done")
		return
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox create --template <template> [<template>...] <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}

	var positionalArgs []string
	var name, fromSnapshot string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from": // means next arg is the snapshot ami id
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --from requires a snapshot AMI ID")
				os.Exit(1)
			}
			i++ // next has to ba a snapshot ami id
			if err := validateSnapshotAmiID(args[i]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fromSnapshot = strings.TrimSpace(args[i])
		default: // means next arg is the template id
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(os.Stderr, "error: %v\n", unknownCreateFlagError(args[i]))
				os.Exit(1)
			}
			arg := strings.TrimSpace(args[i])
			if arg == "" {
				fmt.Fprintln(os.Stderr, "error: template name is required")
				os.Exit(1)
			}
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox create --template <template> [<template>...] <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}

	name = positionalArgs[len(positionalArgs)-1]
	templateRefs := positionalArgs[:len(positionalArgs)-1]

	if len(templateRefs) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one template is required")
		os.Exit(1)
	}
	for _, ref := range templateRefs {
		if strings.HasPrefix(ref, "--") {
			fmt.Fprintln(os.Stderr, "error: template name is required")
			os.Exit(1)
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: devbox create --template <template> [<template>...] <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}
	if strings.HasPrefix(name, "--") {
		fmt.Fprintln(os.Stderr, "error: box name cannot start with --")
		os.Exit(1)
	}

	pubKey := ""
	if pk, err := readPublicKey(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; box will be created without your public key\n", err)
	} else {
		pubKey = pk
	}

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	instanceType := service.DefaultInstanceType
	volumeSizeGB := service.DefaultVolumeSizeGB
	if mode == "local" {
		selected, err := selectInstanceType(service.AllInstanceTypes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error selecting instance type: %v\n", err)
			os.Exit(1)
		}
		if err := service.ValidateInstanceType(selected); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		instanceType = selected

		if fromSnapshot == "" {
			selectedVolume, err := selectVolumeSizeGB()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error selecting volume size: %v\n", err)
				os.Exit(1)
			}
			if err := service.ValidateVolumeSizeGB(selectedVolume); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			volumeSizeGB = selectedVolume
		}
	}

	if fromSnapshot != "" {
		fmt.Printf("Creating box %q from templates %s (snapshot %s)...\n", name, strings.Join(templateRefs, ", "), fromSnapshot)
	} else {
		fmt.Printf("Creating box %q from templates %s...\n", name, strings.Join(templateRefs, ", "))
	}

	var b Box
	if mode == "local" {
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		inst, err := rt.CreateBoxFromTemplates(name, templateRefs, pubKey, fromSnapshot, service.LocalUserID, instanceType, volumeSizeGB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		b = instancesToBoxes([]*service.Instance{inst})[0]
	} else {
		body := map[string]any{"name": name, "templateIds": templateRefs}
		if fromSnapshot != "" {
			body["fromSnapshot"] = fromSnapshot
		}
		if pubKey != "" {
			body["publicKey"] = pubKey
		}

		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Post("/v2/boxes", body)
		if err != nil {
			api.FailBox("create template", err)
		}
		if err := api.CheckStatus(resp); err != nil {
			api.FailBox("create template", err)
		}
		if err := api.DecodeJSON(resp, &b); err != nil {
			api.FailBox("create template", err)
		}
	}

	fmt.Printf("Box created.\n")
	fmt.Printf("  ID:        %s\n", b.ID)
	fmt.Printf("  Name:      %s\n", b.Name)
	fmt.Printf("  Status:    %s\n", b.Status)
	if b.InstanceType != "" {
		fmt.Printf("  Type:      %s\n", b.InstanceType)
	}
	if mode == "local" && fromSnapshot == "" {
		fmt.Printf("  Storage:   %d GB\n", volumeSizeGB)
	}
	if b.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", b.PublicIP)
		fmt.Printf("\n  Connect:   devbox ssh %s\n", b.ID)
	} else {
		fmt.Printf("\n  Provisioning — check status: devbox status %s\n", b.ID)
	}
}
