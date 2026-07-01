package cmd

import (
	"fmt"
	"os"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// Delete permanently deletes a box.
func Delete(args []string) {
	ref := helper.ParseSingleBoxRef(args, "usage: devbox delete <id|name>")

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Are you sure you want to delete box %s (%s)? [y/N] ", target.Name, target.ID)
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "y" {
		fmt.Println("Aborted.")
		return
	}
	if err := rt.DeleteInstance(target.ID, service.LocalUserID); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Box %s (%s) deleted.\n", target.Name, target.ID)

}
