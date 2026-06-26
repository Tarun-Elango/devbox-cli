package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"devbox-cli/service"
)

func Resize(args []string) {
	if TestMode {
		fmt.Println("[test] resize: done")
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devbox resize <id|name>")
		os.Exit(1)
	}
	ref := args[0]

	mode, err := service.EnsureLocalModeAndGetCurrMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if mode != "local" {
		fmt.Fprintln(os.Stderr, "error: resize is only supported in local mode")
		os.Exit(1)
	}

	rt := mustOpenRuntime()
	defer func() { _ = rt.Close() }()
	target, err := resolveBoxTarget(mode, rt, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	info, err := rt.GetResizeInfo(target.ID, service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if info.Instance.Status != "stopped" {
		fmt.Fprintf(os.Stderr, "error: box is %s, not stopped; stop it before resizing\n", info.Instance.Status)
		os.Exit(1)
	}

	newInstanceType := ""
	newVolumeSizeGB := 0

	changeType, err := askYesNo("Change instance type? [y/N] ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if changeType {
		selected, err := selectInstanceTypeWithDefault(service.AllInstanceTypes(), info.Instance.InstanceType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error selecting instance type: %v\n", err)
			os.Exit(1)
		}
		if err := service.ValidateInstanceType(selected); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if selected != info.Instance.InstanceType {
			newInstanceType = selected
		}
	}

	changeSize, err := askYesNo("Change root disk size? This cannot be decreased later. [y/N] ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if changeSize {
		selectedVolume, err := selectVolumeSizeGBWithDefault(info.VolumeSizeGB, info.VolumeSizeGB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error selecting volume size: %v\n", err)
			os.Exit(1)
		}
		if selectedVolume != info.VolumeSizeGB {
			newVolumeSizeGB = selectedVolume
		}
	}

	if newInstanceType == "" && newVolumeSizeGB == 0 {
		fmt.Println("No changes selected.")
		return
	}

	fmt.Printf("\nResize changes for %s (%s):\n", target.Name, target.ID)
	if newInstanceType != "" {
		fmt.Printf("  Type:    %s -> %s\n", info.Instance.InstanceType, newInstanceType)
	}
	if newVolumeSizeGB != 0 {
		fmt.Printf("  Storage: %d GB -> %d GB\n", info.VolumeSizeGB, newVolumeSizeGB)
	}

	ok, err := askYesNo("Apply these changes? [y/N] ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Println("Aborted.")
		return
	}

	updated, err := rt.ResizeInstance(target.ID, service.LocalUserID, newInstanceType, newVolumeSizeGB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Box %s (%s) resized.\n", target.Name, target.ID)
	fmt.Printf("  Status: %s\n", updated.Status)
	if updated.InstanceType != "" {
		fmt.Printf("  Type:   %s\n", updated.InstanceType)
	}
	if newVolumeSizeGB != 0 {
		fmt.Printf("  Storage resize requested: %d GB\n", newVolumeSizeGB)
	}
}

func askYesNo(prompt string) (bool, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.TrimSpace(line)
	return answer == "y" || answer == "Y", nil
}
