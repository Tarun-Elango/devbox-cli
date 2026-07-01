package cmd

import (
	"fmt"
	"os"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// Status displays details for a single box.
func Status(args []string) {

	ref := helper.ParseSingleBoxRef(args, "usage: devbox status <id|name>")

	var b Box

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	inst, err := rt.GetInstance(target.ID, service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	b = instancesToBoxes([]*service.Instance{inst})[0] // from the returned instance, create a Box struct used below

	fmt.Printf("ID:        %s\n", b.ID)
	fmt.Printf("Name:      %s\n", b.Name)
	fmt.Printf("Status:    %s\n", b.Status)
	fmt.Printf("Public IP:  %s\n", b.PublicIP)
	fmt.Printf("Private IP: %s\n", b.PrivateIP)
	fmt.Printf("Type:       %s\n", b.InstanceType)
}
