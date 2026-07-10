package cmd

import (
	"fmt"
	"os"

	"outpost-cli/helper"
	"outpost-cli/service"
)

// Restart reboots a running box.
func Restart(args []string) {

	ref := helper.ParseSingleBoxRef(args, "usage: outpost restart|reboot <id|name>")
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := rt.RebootInstance(target.ID, service.LocalUserID); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Box %s (%s) rebooting.\n", target.Name, target.ID)
}
