package helper

import (
	"fmt"
	"os"

	"outpost-cli/service"
)

// SelectOSFamily prompts for a Linux OS when stdin is a terminal; otherwise returns the default.
// Optional preamble lines are shown above the menu (e.g. to explain why a pick is needed).
func SelectOSFamily(preamble ...string) (string, error) {
	return SelectOSFamilyWithDefault(service.DefaultOSFamily, preamble...)
}

// SelectOSFamilyWithDefault prompts for a Linux OS, highlighting defaultFamily.
func SelectOSFamilyWithDefault(defaultFamily string, preamble ...string) (string, error) {
	profiles := service.AllOSFamilies()
	if !IsTerminal(os.Stdin) {
		for _, line := range preamble {
			fmt.Println(line)
		}
		if len(preamble) > 0 {
			fmt.Printf("  Using default: %s\n", service.MustOSProfile(defaultFamily).DisplayName)
		}
		return defaultFamily, nil
	}

	selected := service.DefaultOSFamilyIndex()
	for i, p := range profiles {
		if p.Family == defaultFamily {
			selected = i
			break
		}
	}

	restore, err := EnableRawMode()
	if err != nil {
		return defaultFamily, nil
	}
	defer restore()

	redraw := func() {
		fmt.Print("\033[H\033[2J")
		for _, line := range preamble {
			fmt.Println(line)
		}
		if len(preamble) > 0 {
			fmt.Println()
		}
		fmt.Println("Select Linux OS (↑/↓, Enter to confirm):")
		fmt.Println()
		for i, p := range profiles {
			prefix := "  "
			if i == selected {
				prefix = "> "
			}
			fmt.Printf("%s%s\n", prefix, p.DisplayName)
		}
	}

	redraw()

	for {
		key, err := ReadKey()
		if err != nil {
			return "", err
		}

		switch key {
		case KeyUp:
			if selected > 0 {
				selected--
				redraw()
			}
		case KeyDown:
			if selected < len(profiles)-1 {
				selected++
				redraw()
			}
		case KeyEnter:
			fmt.Println()
			return profiles[selected].Family, nil
		case KeyCtrlC:
			fmt.Println()
			return "", fmt.Errorf("cancelled")
		}
	}
}
