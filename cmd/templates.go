package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/helper"
	"devbox-cli/service"
)

func TemplateList(args []string) {
	helper.RejectExtraArgs(args, "usage: devbox template")

	var templates []*service.Template
	fmt.Println("Listing local templates")
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	templates, err := rt.ListTemplates(service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
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

	templateRefs, name, fromSnapshot, err := helper.ParseCreateTemplateArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: devbox create --template <template> [<template>...] <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}

	pubKey := ""
	if pk, err := readPublicKey(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; box will be created without your public key\n", err)
	} else {
		pubKey = pk
	}

	volumeSizeGB := service.DefaultVolumeSizeGB
	instanceType, err := helper.SelectInstanceType(service.AllInstanceTypes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error selecting instance type: %v\n", err)
		os.Exit(1)
	}
	if err := service.ValidateInstanceType(instanceType); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fromSnapshot == "" {
		selectedVolume, err := helper.SelectVolumeSizeGB()
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

	if fromSnapshot != "" {
		fmt.Printf("Creating box %q from templates %s (snapshot %s)...\n", name, strings.Join(templateRefs, ", "), fromSnapshot)
	} else {
		fmt.Printf("Creating box %q from templates %s...\n", name, strings.Join(templateRefs, ", "))
	}

	var b Box

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	inst, err := rt.CreateBoxFromTemplates(name, templateRefs, pubKey, fromSnapshot, service.LocalUserID, instanceType, volumeSizeGB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	b = instancesToBoxes([]*service.Instance{inst})[0]

	fmt.Printf("Box created.\n")
	fmt.Printf("  ID:        %s\n", b.ID)
	fmt.Printf("  Name:      %s\n", b.Name)
	fmt.Printf("  Status:    %s\n", b.Status)
	if b.InstanceType != "" {
		fmt.Printf("  Type:      %s\n", b.InstanceType)
	}
	if fromSnapshot == "" {
		fmt.Printf("  Storage:   %d GB\n", volumeSizeGB)
	}
	if b.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", b.PublicIP)
		fmt.Printf("\n  Connect:   devbox ssh %s\n", b.ID)
	} else {
		fmt.Printf("\n  Provisioning — check status: devbox status %s\n", b.ID)
	}
}
