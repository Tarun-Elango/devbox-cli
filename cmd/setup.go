package cmd

import (
	"fmt"
	"os"

	"devbox-cli/helper"
	"devbox-cli/service"
)

// setupExit is os.Exit by default; tests replace it to capture exit codes.
var setupExit = os.Exit

// Setup prompts for AWS secret, access key, and region, then saves to ~/.devbox/.
func Setup(args []string) {
	helper.RejectExtraArgs(args, "usage: devbox setup")

	fmt.Println("Setup AWS credentials, if you have already done this, doing this will overwrite your existing credentials, CTRL+C to cancel.")

	secret, err := helper.ReadPassword("AWS secret access key: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading secret: %v\n", err)
		setupExit(1)
	}
	if secret == "" {
		fmt.Fprintln(os.Stderr, "setup failed: secret is required")
		setupExit(1)
	}

	accessKey, err := helper.ReadPassword("AWS access key ID: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading access key: %v\n", err)
		setupExit(1)
	}
	if accessKey == "" {
		fmt.Fprintln(os.Stderr, "setup failed: access key is required")
		setupExit(1)
	}

	regions := service.AllRegions() // get all regions
	region, err := selectRegion(regions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error selecting region: %v\n", err)
		setupExit(1)
	}

	if err := service.SaveAWSCredentials(secret, accessKey, region); err != nil {
		fmt.Fprintf(os.Stderr, "save config: %v\n", err)
		setupExit(1)
	}
}

// ClearCreds prompts for confirmation, then removes saved AWS credentials.
func ClearCreds(args []string) {
	helper.RejectExtraArgs(args, "usage: devbox clear-creds")

	fmt.Print("Are you sure you want to clear saved AWS credentials? [y/N] ")
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "y" && answer != "Y" {
		fmt.Println("Aborted.")
		return
	}

	if err := service.ClearAWSCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "clear credentials: %v\n", err)
		setupExit(1)
	}
}

// function to select region
func selectRegion(regions []service.Region) (string, error) {
	if !helper.IsTerminal(os.Stdin) {
		return selectRegionFallback(regions)
	}

	selected := 0
	const visible = 12

	restore, err := helper.EnableRawMode()
	if err != nil {
		return selectRegionFallback(regions)
	}
	defer restore()

	redraw := func() {
		fmt.Print("\033[H\033[2J")
		fmt.Println("Select AWS region (↑/↓, Enter to confirm):")
		fmt.Println()

		start := selected - visible/2
		if start < 0 {
			start = 0
		}
		end := start + visible
		if end > len(regions) {
			end = len(regions)
			start = end - visible
			if start < 0 {
				start = 0
			}
		}

		for i := start; i < end; i++ {
			prefix := "  "
			if i == selected {
				prefix = "> "
			}
			fmt.Printf("%s%s  %s\n", prefix, regions[i].ID, regions[i].Name)
		}
	}

	redraw()

	for {
		key, err := helper.ReadKey()
		if err != nil {
			return "", err
		}

		switch key {
		case helper.KeyUp:
			if selected > 0 {
				selected--
				redraw()
			}
		case helper.KeyDown:
			if selected < len(regions)-1 {
				selected++
				redraw()
			}
		case helper.KeyEnter:
			fmt.Println()
			return regions[selected].ID, nil
		case helper.KeyCtrlC:
			fmt.Println()
			return "", fmt.Errorf("cancelled")
		}
	}
}

func selectRegionFallback(regions []service.Region) (string, error) {
	fmt.Println("Select AWS region:")
	for i, r := range regions {
		fmt.Printf("  %2d) %s  %s\n", i+1, r.ID, r.Name)
	}
	fmt.Print("Enter number or region id: ")

	line, err := helper.ReadStdinLine()
	if err != nil {
		return "", err
	}

	if n := 0; len(line) > 0 {
		if _, err := fmt.Sscanf(line, "%d", &n); err == nil {
			if n >= 1 && n <= len(regions) {
				return regions[n-1].ID, nil
			}
		}
	}
	for _, r := range regions {
		if r.ID == line {
			return r.ID, nil
		}
	}
	return "", fmt.Errorf("invalid region %q", line)
}
