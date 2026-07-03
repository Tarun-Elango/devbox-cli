package cmd

import (
	"fmt"
	"os"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// Create creates a new box with an optional name and returns as soon as EC2 accepts the launch.
// Pass --from <amiId|name> to restore from a previously saved snapshot.
func Create(args []string) {
	if len(args) > 0 && args[0] == "--template" {
		CreateTemplate(args[1:])
		return
	}

	name, fromSnapshot, err := helper.ParseNameAndFromFlag(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: devbox create <name> [--from <amiId|name>]")
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
	fmt.Printf("  SSH config: devbox-%s added to ~/.ssh/config\n", b.Name)
	if b.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", b.PublicIP)
	} else {
		fmt.Printf("\nThe box is still provisioning.\n")
	}

	fmt.Printf("\nRecommended next steps:\n")
	fmt.Printf("  1. Check box status: devbox status %s\n", b.ID)
	fmt.Printf("  2. Wait for SSH and template setup: devbox ssh %s\n", b.ID)
	fmt.Printf("     Use this command to check SSH/template readiness and connect once the box is ready.\n")
	fmt.Printf("  3. After devbox ssh %s succeeds, you can also connect outside this CLI with:\n", b.ID)
	fmt.Printf("     ssh devbox-%s\n", b.Name)
	fmt.Printf("     VS Code Remote-SSH -> devbox-%s\n", b.Name)
}
