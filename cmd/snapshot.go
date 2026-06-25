package cmd

import (
	"fmt"
	"os"

	"devbox-cli/internal/api"
	"devbox-cli/service"
)

// Snapshot creates an AMI snapshot of the given box.
// Usage: devbox snapshot <id|name> [name]
func Snapshot(args []string) {
	if TestMode {
		fmt.Println("[test] snapshot: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox snapshot <id|name> [name]")
		os.Exit(1)
	}
	ref := args[0]
	name := "snapshot-" + ref
	if len(args) >= 2 {
		name = args[1]
	}

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var result service.Snapshot
	if mode == "local" {
		rt := mustOpenRuntime()
		defer func() { _ = rt.Close() }()
		target, err := resolveBoxTarget(mode, rt, ref)
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
	} else {
		target, err := resolveBoxTarget(mode, nil, ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
			os.Exit(1)
		}
		client, err := api.NewDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		body := map[string]string{"name": name}
		resp, err := client.Post("/v1/boxes/"+target.ID+"/snapshots", body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
			os.Exit(1)
		}
		if err := api.CheckStatus(resp); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
			os.Exit(1)
		}
		if err := api.DecodeJSON(resp, &result); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Snapshot created: %s  name=%s  state=%s\n", result.AmiID, result.Name, result.State)
	fmt.Printf("Snapshot creation may take a minute or so to finish. Check status with: devbox snapshots ls %s\n", result.AmiID)
}
