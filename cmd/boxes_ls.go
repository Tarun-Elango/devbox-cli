package cmd

import (
	"fmt"
	"os"
	"strings"

	"outpost-cli/helper"
	"outpost-cli/service"
)

// Ls lists all boxes belonging to the authenticated user.
func Ls(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost ls")
	var boxes []Box

	fmt.Println("Listing local boxes")
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	instances, err := rt.ListInstances(service.LocalUserID)
	if err != nil && len(instances) == 0 {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	boxes = instancesToBoxes(instances)

	if len(boxes) == 0 {
		if err != nil {
			return
		}
		fmt.Println("No boxes found.")
		return
	}

	fmt.Printf("%-24s  %-20s  %-10s  %-8s  %-16s  %-16s\n", "ID", "NAME", "STATUS", "PROVIDER", "REGION", "PUBLIC IP")
	fmt.Println(strings.Repeat("-", 100))
	for _, b := range boxes {
		fmt.Printf("%-24s  %-20s  %-10s  %-8s  %-16s  %-16s\n", b.ID, b.Name, b.Status, b.Provider, b.Region, b.PublicIP)
	}
}
