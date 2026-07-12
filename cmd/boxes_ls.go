package cmd

import (
	"fmt"
	"os"

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

	printBoxTable(boxes, helper.StdoutWidth())
}

func printBoxTable(boxes []Box, termWidth int) {
	headers := []string{"ID", "NAME", "STATUS", "OS", "PROVIDER", "REGION", "PUBLIC IP"}
	preferred := []int{24, 18, 10, 14, 8, 14, 16}
	min := []int{12, 8, 6, 8, 3, 8, 7}

	rows := make([][]string, len(boxes))
	for i, b := range boxes {
		osLabel := b.OSFamily
		if p, ok := service.OSProfileFor(b.OSFamily); ok {
			osLabel = p.DisplayName
		}
		rows[i] = []string{b.ID, b.Name, b.Status, osLabel, b.Provider, b.Region, b.PublicIP}
	}
	printTable(headers, rows, preferred, min, termWidth)
}
