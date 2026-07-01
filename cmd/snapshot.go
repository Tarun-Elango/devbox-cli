package cmd

import (
	"fmt"
	"os"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// Snapshot creates an AMI snapshot of the given box.
// Usage: devbox snapshot <id|name> <name>
func Snapshot(args []string) {

	ref, name := helper.ParseSnapshotArgs(args, "usage: devbox snapshot <id|name> <name>")
	if name == "" {
		fmt.Fprintln(os.Stderr, "error: snapshot name is required")
		os.Exit(1)
	}

	var result service.Snapshot
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveBoxTarget(rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}
	snap, err := rt.CreateSnapshot(target.ID, name, service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}
	result = *snap

	fmt.Printf("Snapshot created: %s  name=%s  state=%s\n", result.AmiID, result.Name, result.State)
	fmt.Printf("Snapshot creation may take a minute or so to finish. Check status with: devbox snapshots ls %s\n", result.AmiID)
}
