package cmd

import (
	"fmt"
	"os"
	"strings"

	"outpost-cli/helper"
	"outpost-cli/service"
)

const snapshotUsage = "usage: outpost snapshot [ls [<amiId|name>]] | create <id|name> <name> | delete <amiId|name>"

// snapshotItem represents a snapshot as returned by the API.
type snapshotItem struct {
	AmiID    string `json:"amiId"`
	Name     string `json:"name"`
	State    string `json:"state"`
	BoxAwsID string `json:"boxAwsId"`
	Region   string `json:"region"`
	Provider string `json:"provider"`
}

// Snapshot dispatches snapshot sub-commands.
//
//	outpost snapshot [ls]                    → list all user snapshots
//	outpost snapshot ls <amiId|name>         → show details for a specific snapshot
//	outpost snapshot create <id|name> <name> → create a snapshot
//	outpost snapshot delete <amiId|name>     → delete a snapshot
func Snapshot(args []string) {
	if len(args) == 0 {
		snapshotsList(args)
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "ls":
		if len(subArgs) == 0 {
			snapshotsList(subArgs)
		} else {
			ref := helper.ParseSingleSnapshotRefArg(subArgs, "usage: outpost snapshot ls [<amiId|name>]")
			snapshotShowByRef(ref)
		}
	case "create":
		snapshotCreate(subArgs)
	case "delete":
		ref := helper.ParseSingleSnapshotRefArg(subArgs, "usage: outpost snapshot delete <amiId|name>")
		snapshotDeleteByRef(ref)
	default:
		fmt.Fprintf(os.Stderr, "snapshot: unknown sub-command %q\n", sub)
		fmt.Fprintln(os.Stderr, snapshotUsage)
		os.Exit(1)
	}
}

func snapshotCreate(args []string) {
	ref, name := helper.ParseSnapshotArgs(args, "usage: outpost snapshot create <id|name> <name>")
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
	fmt.Printf("Snapshot creation may take a minute or so to finish. Check status with: outpost snapshot ls %s\n", result.Name)
}

func snapshotShowByRef(ref string) {
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveSnapshotTarget(rt, ref)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "snapshot %s not found\n", ref)
		} else {
			fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		}
		os.Exit(1)
	}
	snapshotsShow(target.AmiID)
}

func snapshotDeleteByRef(ref string) {
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := helper.ResolveSnapshotTarget(rt, ref)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "snapshot %s not found\n", ref)
		} else {
			fmt.Fprintf(os.Stderr, "snapshot delete failed: %v\n", err)
		}
		os.Exit(1)
	}
	snapshotsDelete(target.AmiID)
}

func snapshotsList(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost snapshot [ls]")

	var items []snapshotItem
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	snaps, err := rt.ListSnapshots(service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}
	items = snapshotsToItems(snaps)

	if len(items) == 0 {
		fmt.Println("No snapshots found.")
		return
	}

	printSnapshotTable(items)
}

func snapshotsToItems(snaps []*service.Snapshot) []snapshotItem {
	items := make([]snapshotItem, len(snaps))
	for i, s := range snaps {
		items[i] = snapshotItem{
			AmiID:    s.AmiID,
			Name:     s.Name,
			State:    s.State,
			BoxAwsID: s.BoxAwsID,
			Region:   s.Region,
			Provider: s.Provider,
		}
	}
	return items
}

func snapshotsShow(amiID string) {
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	snap, err := rt.GetSnapshot(amiID, service.LocalUserID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "snapshot %s not found\n", amiID)
		} else {
			fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		}
		os.Exit(1)
	}

	printSnapshotTable(snapshotsToItems([]*service.Snapshot{snap}))
}

func snapshotsDelete(amiID string) {
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	if err := rt.DeleteSnapshot(amiID, service.LocalUserID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "snapshot %s not found\n", amiID)
		} else {
			fmt.Fprintf(os.Stderr, "snapshot delete failed: %v\n", err)
		}
		os.Exit(1)
	}
	fmt.Printf("Snapshot %s deleted.\n", amiID)
}

func printSnapshotTable(items []snapshotItem) {
	fmt.Printf("%-24s  %-20s  %-12s  %-14s  %-10s  %s\n", "AMI ID", "NAME", "STATE", "REGION", "PROVIDER", "BOX ID")
	fmt.Println(strings.Repeat("-", 120))
	for _, s := range items {
		fmt.Printf("%-24s  %-20s  %-12s  %-14s  %-10s  %s\n", s.AmiID, s.Name, s.State, s.Region, s.Provider, s.BoxAwsID)
	}
}
