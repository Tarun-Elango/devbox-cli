package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"devbox-cli/internal/api"
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
	if TestMode {
		fmt.Println("[test] snapshots: done")
		return
	}

	if len(args) == 0 {
		snapshotsList()
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "ls":
		if len(subArgs) < 1 {
			fmt.Fprintln(os.Stderr, "usage: devbox snapshots ls <amiId>")
			os.Exit(1)
		}
		snapshotsShow(subArgs[0])
	case "delete":
		if len(subArgs) < 1 {
			fmt.Fprintln(os.Stderr, "usage: devbox snapshots delete <amiId>")
			os.Exit(1)
		}
		snapshotsDelete(subArgs[0])
	default:
		fmt.Fprintf(os.Stderr, "snapshots: unknown sub-command %q\n", sub)
		fmt.Fprintln(os.Stderr, "usage: devbox snapshots [ls <amiId> | delete <amiId>]")
		os.Exit(1)
	}
}

func snapshotsList() {
	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var items []snapshotItem
	if mode == "local" {
		snaps, err := service.ListSnapshots(service.LocalUserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "snapshots failed: %v\n", err)
			os.Exit(1)
		}
		items = snapshotsToItems(snaps)
	} else {
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Get("/v1/snapshots")
		if err != nil {
			fmt.Fprintf(os.Stderr, "snapshots failed: %v\n", err)
			os.Exit(1)
		}
		if err := api.CheckStatus(resp); err != nil {
			fmt.Fprintf(os.Stderr, "snapshots failed: %v\n", err)
			os.Exit(1)
		}

		if err := api.DecodeJSON(resp, &items); err != nil { // add response to items
			fmt.Fprintf(os.Stderr, "snapshots failed: %v\n", err)
			os.Exit(1)
		}
	}

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
	if _, err := service.EnsureLocalModeAndGetCurrMode(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	snap, err := service.GetSnapshot(amiID, service.LocalUserID)
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
	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if mode == "local" {
		if err := service.DeleteSnapshot(amiID, service.LocalUserID); err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Fprintf(os.Stderr, "snapshot %s not found\n", amiID)
			} else {
				fmt.Fprintf(os.Stderr, "snapshot delete failed: %v\n", err)
			}
			os.Exit(1)
		}
		fmt.Printf("Snapshot %s deleted.\n", amiID)
		return
	}

	client, err := api.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Delete("/v1/snapshots/" + amiID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot delete failed: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		fmt.Fprintf(os.Stderr, "snapshot %s not found\n", amiID)
		os.Exit(1)
	}

	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "snapshot delete failed: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()

	fmt.Printf("Snapshot %s deleted.\n", amiID)
}

func printSnapshotTable(items []snapshotItem) {
	fmt.Printf("%-24s  %-20s  %-12s  %s\n", "AMI ID", "NAME", "STATE", "BOX ID")
	fmt.Println(strings.Repeat("-", 90))
	for _, s := range items {
		fmt.Printf("%-24s  %-20s  %-12s  %s\n", s.AmiID, s.Name, s.State, s.BoxAwsID)
	}
}
