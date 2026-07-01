package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// snapshotItem represents a snapshot as returned by the API.
type snapshotItem struct {
	AmiID    string `json:"amiId"`
	Name     string `json:"name"`
	State    string `json:"state"`
	BoxAwsID string `json:"boxAwsId"`
}

// Snapshots dispatches snapshot sub-commands.
//
//	devbox snapshots              → list all user snapshots
//	devbox snapshots ls <amiId>   → show details for a specific snapshot
//	devbox snapshots delete <amiId> → delete a snapshot
func Snapshots(args []string) {

	if len(args) == 0 {
		snapshotsList(args)
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "ls":
		amiID := helper.ParseSingleSnapshotAmiIDArg(subArgs, "usage: devbox snapshots ls <amiId>")
		snapshotsShow(amiID)
	case "delete":
		amiID := helper.ParseSingleSnapshotAmiIDArg(subArgs, "usage: devbox snapshots delete <amiId>")
		snapshotsDelete(amiID)
	default:
		fmt.Fprintf(os.Stderr, "snapshots: unknown sub-command %q\n", sub)
		fmt.Fprintln(os.Stderr, "usage: devbox snapshots [ls <amiId> | delete <amiId>]")
		os.Exit(1)
	}
}

func snapshotsList(args []string) {
	helper.RejectExtraArgs(args, "usage: devbox snapshots")

	var items []snapshotItem
	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	snaps, err := rt.ListSnapshots(service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshots failed: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "snapshots failed: %v\n", err)
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
	fmt.Printf("%-24s  %-20s  %-12s  %s\n", "AMI ID", "NAME", "STATE", "BOX ID")
	fmt.Println(strings.Repeat("-", 90))
	for _, s := range items {
		fmt.Printf("%-24s  %-20s  %-12s  %s\n", s.AmiID, s.Name, s.State, s.BoxAwsID)
	}
}
