// this helps users set mode cloud or local, and add to .devbox/config.json
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"devbox-cli/internal/config"
)

type modeOption struct {
	ID          string
	Description string
}

var modeOptions = []modeOption{
	{ID: "local", Description: "bring your own key and secret"},
	{ID: "cloud", Description: "fully managed need to login"},
}

func Mode(args []string) {
	if TestMode {
		fmt.Println("[test] mode: done")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	selected, err := selectMode(cfg.Mode) // selects the users choice of mode
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg.Mode = selected // sets the mode to the users choice
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Mode set to %s.\n", selected)
}

func selectMode(current string) (string, error) {
	if !isTerminal(os.Stdin) {
		return selectModeFallback(current)
	}

	selected := 0
	for i, opt := range modeOptions { // find what mode the user is currently on
		if opt.ID == current {
			selected = i
			break
		}
	}

	restore, err := enableRawMode() // enables raw mode for the terminal
	if err != nil {
		return selectModeFallback(current)
	}
	defer restore()

	statusLine := "mode not set yet"
	if current != "" {
		statusLine = fmt.Sprintf("current mode is %s", current)
	}

	menuLines := 2 + len(modeOptions) // header, blank, one line per option
	drawn := false

	redraw := func() {
		if drawn {
			fmt.Printf("\033[%dA", menuLines)
		}
		fmt.Printf("\033[2K\r%s\n", "Select mode (↑/↓, Enter to confirm):")
		fmt.Print("\033[2K\r\n")
		for i, opt := range modeOptions {
			prefix := "  "
			if i == selected {
				prefix = "> "
			}
			fmt.Printf("\033[2K\r%s%s  %s\n", prefix, opt.ID, opt.Description)
		}
		drawn = true
	}

	fmt.Printf("%s\n\n", statusLine)
	redraw()

	// infinite loop to allow the user to select the mode
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
			if selected < len(modeOptions)-1 {
				selected++
				redraw()
			}
		case keyEnter:
			fmt.Println()
			return modeOptions[selected].ID, nil
		case keyCtrlC:
			fmt.Println()
			return "", fmt.Errorf("cancelled")
		}
	}
}

func selectModeFallback(current string) (string, error) {
	fmt.Println("Select mode:")
	for i, opt := range modeOptions {
		marker := " "
		if opt.ID == current {
			marker = "*"
		}
		fmt.Printf("  %s %d) %s  %s\n", marker, i+1, opt.ID, opt.Description)
	}
	fmt.Print("Enter number or mode name: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)

	if n := 0; len(line) > 0 {
		if _, err := fmt.Sscanf(line, "%d", &n); err == nil {
			if n >= 1 && n <= len(modeOptions) {
				return modeOptions[n-1].ID, nil
			}
		}
	}
	for _, opt := range modeOptions {
		if opt.ID == line {
			return opt.ID, nil
		}
	}
	return "", fmt.Errorf("invalid mode %q", line)
}


