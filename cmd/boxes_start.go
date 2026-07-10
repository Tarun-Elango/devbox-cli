package cmd

import (
	"fmt"
	"os"

	"outpost-cli/helper"
	"outpost-cli/service"
)

// Start starts a stopped box.
func Start(args []string) {

	ref := helper.ParseSingleBoxRef(args, "usage: outpost start <id|name>")

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := rt.StartInstance(target.ID, service.LocalUserID); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Box %s (%s) started.\n", target.Name, target.ID)

}
