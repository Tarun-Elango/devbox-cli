package cmd

import (
	"fmt"
	"os"

	"devbox-cli/service"
)

func selectInstanceType(types []service.InstanceType) (string, error) {
	return selectInstanceTypeWithDefault(types, service.DefaultInstanceType)
}

func selectInstanceTypeWithDefault(types []service.InstanceType, defaultType string) (string, error) {
	if !isTerminal(os.Stdin) {
		return defaultType, nil
	}

	selected := service.DefaultInstanceTypeIndex()
	for i, t := range types {
		if t.ID == defaultType {
			selected = i
			break
		}
	}
	const visible = 12

	restore, err := enableRawMode()
	if err != nil {
		return defaultType, nil
	}
	defer restore()

	redraw := func() {
		fmt.Print("\033[H\033[2J")
		fmt.Println("Select instance type (↑/↓, Enter to confirm):")
		fmt.Println()

		start := selected - visible/2
		if start < 0 {
			start = 0
		}
		end := start + visible
		if end > len(types) {
			end = len(types)
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
			fmt.Printf("%s%s  %s\n", prefix, types[i].ID, types[i].Label)
		}
	}

	redraw()

	for {
		key, err := readKey()
		if err != nil {
			return "", err
		}

		switch key {
		case keyUp:
			if selected > 0 {
				selected--
				redraw()
			}
		case keyDown:
			if selected < len(types)-1 {
				selected++
				redraw()
			}
		case keyEnter:
			fmt.Println()
			return types[selected].ID, nil
		case keyCtrlC:
			fmt.Println()
			return "", fmt.Errorf("cancelled")
		}
	}
}
