package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/internal/api"
	"devbox-cli/service"
)

// readPublicKey returns the contents of ~/.ssh/id_ed25519.pub, prompting to
// create the key pair with ssh-keygen when it is missing.
func readPublicKey() (string, error) {
	_, pub, err := ed25519KeyPaths() // get the path to the public key
	if err != nil {
		return "", err
	}

	if data, err := os.ReadFile(pub); err == nil {
		return strings.TrimSpace(string(data)), nil // if exists, return the public key
	}

	fmt.Printf("No SSH public key found at %s. Create one now? [y/N] ", pub)
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "y" && answer != "Y" {
		return "", fmt.Errorf("no public key at %s", pub)
	}

	if err := ensureEd25519Key(); err != nil { // create the key pair if it is missing
		return "", err
	}

	// if successful created, print success message
	fmt.Printf("SSH public key created in your ~/.ssh directory.")

	data, err := os.ReadFile(pub) // read the public key file
	if err != nil {
		return "", fmt.Errorf("read %s: %w", pub, err)
	}
	return strings.TrimSpace(string(data)), nil // return the public key
}

// Box represents a devbox instance as returned by the API.
type Box struct {
	ID           string `json:"instanceId"`
	Name         string `json:"name"`
	Status       string `json:"state"`
	InstanceType string `json:"instanceType"`
	PublicIP     string `json:"publicIpAddress"`
	PrivateIP    string `json:"privateIpAddress"`
}

func instancesToBoxes(instances []*service.Instance) []Box {
	boxes := make([]Box, len(instances))
	for i, inst := range instances {
		boxes[i] = Box{
			ID:           inst.ID,
			Name:         inst.Name,
			Status:       inst.Status,
			InstanceType: inst.InstanceType,
			PublicIP:     inst.IPAddress,
			PrivateIP:    inst.PrivateIPAddress,
		}
	}
	return boxes
}

// Ls lists all boxes belonging to the authenticated user.
func Ls() {
	if TestMode {
		fmt.Println("[test] ls: done")
		return
	}

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var boxes []Box
	if mode == "local" {
		fmt.Println("Listing local boxes")
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		instances, err := rt.ListInstances(service.LocalUserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		boxes = instancesToBoxes(instances)
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Get("/v1/boxes")
		if err != nil {
			api.FailBox("ls", err)
		}
		if err := api.CheckStatus(resp); err != nil {
			api.FailBox("ls", err)
		}

		if err := api.DecodeJSON(resp, &boxes); err != nil {
			api.FailBox("ls", err)
		}
	}

	if len(boxes) == 0 {
		fmt.Println("No boxes found.")
		return
	}

	fmt.Printf("%-24s  %-20s  %-10s  %-16s\n", "ID", "NAME", "STATUS", "PUBLIC IP")
	fmt.Println(strings.Repeat("-", 80))
	for _, b := range boxes {
		fmt.Printf("%-24s  %-20s  %-10s  %-16s\n", b.ID, b.Name, b.Status, b.PublicIP)
	}
}

// Status displays details for a single box.
func Status(args []string) {
	if TestMode {
		fmt.Println("[test] status: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox status <id>")
		os.Exit(1)
	}
	id := args[0]

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var b Box
	if mode == "local" {
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		inst, err := rt.GetInstance(id, service.LocalUserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		b = instancesToBoxes([]*service.Instance{inst})[0] // from the returned instance, create a Box struct used below
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Get("/v1/boxes/" + id)
		if err != nil {
			api.FailBox("status", err)
		}
		if err := api.CheckStatus(resp); err != nil {
			api.FailBox("status", err)
		}

		if err := api.DecodeJSON(resp, &b); err != nil {
			api.FailBox("status", err)
		}
	}

	fmt.Printf("ID:        %s\n", b.ID)
	fmt.Printf("Name:      %s\n", b.Name)
	fmt.Printf("Status:    %s\n", b.Status)
	fmt.Printf("Public IP:  %s\n", b.PublicIP)
	fmt.Printf("Private IP: %s\n", b.PrivateIP)
	fmt.Printf("Type:       %s\n", b.InstanceType)
}

// Create creates a new box with an optional name and returns as soon as EC2 accepts the launch.
// Pass --from <snapshot_ami_id> to restore from a previously saved snapshot.
func Create(args []string) {
	if len(args) > 0 && args[0] == "--template" {
		CreateTemplate(args[1:])
		return
	}

	if TestMode {
		fmt.Println("[test] create: done")
		return
	}

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	name, fromSnapshot, err := ParseNameAndFromFlag(args) // should have at least name
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: devbox create <name> [--from <snapshot_ami_id>]")
		os.Exit(1)
	}

	pubKey := ""
	if pk, err := readPublicKey(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; box will be created without your public key\n", err)
	} else {
		pubKey = pk
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
		fmt.Printf("Creating box %q from snapshot AMI %s...\n", name, fromSnapshot)
	} else {
		fmt.Printf("Creating box %q...\n", name)
	}

	var b Box
	if mode == "local" {
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		inst, err := rt.CreateInstance(name, pubKey, fromSnapshot, service.LocalUserID, instanceType, volumeSizeGB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		b = instancesToBoxes([]*service.Instance{inst})[0]
	} else {
		body := map[string]string{"name": name}
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
			api.FailBox("create", err)
		}
		if err := api.CheckStatus(resp); err != nil {
			api.FailBox("create", err)
		}

		if err := api.DecodeJSON(resp, &b); err != nil {
			api.FailBox("create", err)
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

// Stop stops a running box.
func Stop(args []string) {
	if TestMode {
		fmt.Println("[test] stop: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox stop <id>")
		os.Exit(1)
	}
	id := args[0]

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if mode == "local" {
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		if err := rt.StopInstance(id, service.LocalUserID); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Post("/v1/boxes/"+id+"/stop", nil)
		if err != nil {
			api.FailBox("stop", err)
		}
		if err := api.CheckStatus(resp); err != nil {
			api.FailBox("stop", err)
		}
		if err := resp.Body.Close(); err != nil {
			api.FailBox("stop", err)
		}
	}

	fmt.Printf("Box %s stopped.\n", id)
}

// Start starts a stopped box.
func Start(args []string) {
	if TestMode {
		fmt.Println("[test] start: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox start <id>")
		os.Exit(1)
	}
	id := args[0]

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if mode == "local" {
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		if err := rt.StartInstance(id, service.LocalUserID); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Post("/v1/boxes/"+id+"/start", nil)
		if err != nil {
			api.FailBox("start", err)
		}
		if err := api.CheckStatus(resp); err != nil {
			api.FailBox("start", err)
		}
		if err := resp.Body.Close(); err != nil {
			api.FailBox("start", err)
		}
	}

	fmt.Printf("Box %s started.\n", id)
}

// Delete permanently deletes a box.
func Delete(args []string) {
	if TestMode {
		fmt.Println("[test] delete: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox delete <id>")
		os.Exit(1)
	}
	id := args[0]

	fmt.Printf("Are you sure you want to delete box %s? [y/N] ", id)
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "y" {
		fmt.Println("Aborted.")
		return
	}

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if mode == "local" {
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		if err := rt.DeleteInstance(id, service.LocalUserID); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Delete("/v1/boxes/" + id)
		if err != nil {
			api.FailBox("delete", err)
		}
		if err := api.CheckStatus(resp); err != nil {
			api.FailBox("delete", err)
		}
		if err := resp.Body.Close(); err != nil {
			api.FailBox("delete", err)
		}
	}

	fmt.Printf("Box %s deleted.\n", id)
}
