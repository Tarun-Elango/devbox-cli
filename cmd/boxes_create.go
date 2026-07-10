package cmd

import (
	"fmt"
	"os"

	"outpost-cli/helper"
	"outpost-cli/service"
)

// Create creates a new box with a name and returns as soon as EC2 accepts the launch.
// Pass --template <templateName>... to apply one or more templates' startup scripts.
// Pass --from <amiId|name> to restore from a previously saved snapshot.
func Create(args []string) {
	name, templateRefs, fromSnapshot, err := helper.ParseCreateArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: outpost create <name> [--template <templateName>...] [--from <amiId|name>]")
		os.Exit(1)
	}

	if len(templateRefs) > 0 {
		createFromTemplates(name, templateRefs, fromSnapshot)
		return
	}

	pubKey, err := readPublicKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
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
		fmt.Printf("Creating box %q from snapshot %s...\n", name, fromSnapshot)
	} else {
		fmt.Printf("Creating box %q...\n", name)
	}

	var b Box
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	if fromSnapshot != "" {
		snapshotTarget, err := helper.ResolveSnapshotTarget(rt, fromSnapshot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fromSnapshot = snapshotTarget.AmiID
	}

	inst, err := rt.CreateInstance(name, pubKey, fromSnapshot, service.LocalUserID, instanceType, volumeSizeGB)
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
	addSSHHostOrWarn(b.Name, inst)
	if b.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", b.PublicIP)
	} else {
		fmt.Printf("\nThe box is still provisioning.\n")
	}

	fmt.Printf("\nRecommended next steps:\n")
	fmt.Printf("  1. Check box status: outpost status %s\n", b.Name)
	fmt.Printf("  2. Wait for SSH and template setup: outpost ssh %s\n", b.Name)
	fmt.Printf("     Use this command to check SSH/template readiness and connect once the box is ready.\n")
	fmt.Printf("  3. After outpost ssh %s succeeds, you can also connect outside this CLI with:\n", b.Name)
	fmt.Printf("     ssh outpost-%s\n", b.Name)
	fmt.Printf("     VS Code Remote-SSH -> outpost-%s\n", b.Name)
}
