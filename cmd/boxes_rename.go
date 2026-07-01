package cmd

import (
	"fmt"
	"os"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// Rename updates a local box name in AWS, the local DB, and SSH config.
func Rename(args []string) {
	ref, newName := helper.ParseRenameBoxArgs(args, "usage: devbox rename <id|name> <new-name>")

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	renamed, err := rt.RenameInstance(target.ID, service.LocalUserID, newName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Box %s (%s) renamed to %s.\n", target.Name, target.ID, renamed.Name)
	fmt.Printf("SSH config: devbox-%s updated to devbox-%s\n", target.Name, renamed.Name)
}
