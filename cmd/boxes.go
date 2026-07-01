package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/service"
)

// mostly just util and helper functions for the boxes command

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
