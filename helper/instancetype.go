package helper

import (
	"fmt"
	"os"

	"outpost-cli/service"
)

func SelectInstanceType(types []service.InstanceType) (string, error) {
	return SelectInstanceTypeWithDefault(types, service.DefaultInstanceType)
}

func SelectInstanceTypeWithDefault(types []service.InstanceType, defaultType string) (string, error) {
	if !IsTerminal(os.Stdin) {
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

	restore, err := EnableRawMode()
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
			fmt.Printf("%s%s\n", prefix, types[i].Label)
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
			if selected < len(types)-1 {
				selected++
				redraw()
			}
		case KeyEnter:
			fmt.Println()
			return types[selected].ID, nil
		case KeyCtrlC:
			fmt.Println()
			return "", fmt.Errorf("cancelled")
		}
	}
}
