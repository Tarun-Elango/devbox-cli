package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// Ls lists all boxes belonging to the authenticated user.
func Ls(args []string) {
	helper.RejectExtraArgs(args, "usage: devbox ls")
	var boxes []Box

	fmt.Println("Listing local boxes")
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	instances, err := rt.ListInstances(service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	boxes = instancesToBoxes(instances)

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
