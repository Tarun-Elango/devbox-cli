package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"devbox-cli/service"
)

// Setup prompts for AWS secret, access key, and region, then saves to ~/.devbox/.
func Setup(args []string) {
	if TestMode {
		fmt.Println("[test] setup: done")
		return
	}
	fmt.Println("Setup AWS credentials, if you have already done this, doing this will overwrite your existing credentials, CTRL+C to cancel.")

	secret, err := readPassword("AWS secret access key: ") // from auth.go
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading secret: %v\n", err)
		os.Exit(1)
	}
	if secret == "" {
		fmt.Fprintln(os.Stderr, "setup failed: secret is required")
		os.Exit(1)
	}

	accessKey, err := readPassword("AWS access key ID: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading access key: %v\n", err)
		os.Exit(1)
	}
	if accessKey == "" {
		fmt.Fprintln(os.Stderr, "setup failed: access key is required")
		os.Exit(1)
	}

	regions := service.AllRegions()// get all regions
	region, err := selectRegion(regions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error selecting region: %v\n", err)
		os.Exit(1)
	}

	if err := service.SaveAWSCredentials(secret, accessKey, region); err != nil {
		fmt.Fprintf(os.Stderr, "save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("AWS credentials saved (region: %s).\n", region)
}

// function to select region
func selectRegion(regions []service.Region) (string, error) {
	if !isTerminal(os.Stdin) {
		return selectRegionFallback(regions)
	}

	selected := 0
	const visible = 12

	restore, err := enableRawMode()
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
			if selected < len(regions)-1 {
				selected++
				redraw()
			}
		case keyEnter:
			fmt.Println()
			return regions[selected].ID, nil
		case keyCtrlC:
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

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)

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

type keyCode int

const (
	keyUp keyCode = iota
	keyDown
	keyEnter
	keyCtrlC
	keyOther
)

func readKey() (keyCode, error) {
	var b [1]byte
	if _, err := os.Stdin.Read(b[:]); err != nil {
		return keyOther, err
	}

	switch b[0] {
	case 3: // Ctrl+C
		return keyCtrlC, nil
	case '\r', '\n':
		return keyEnter, nil
	case 27: // ESC — arrow keys
		seq := make([]byte, 2)
		if _, err := os.Stdin.Read(seq[:]); err != nil {
			return keyOther, err
		}
		if seq[0] != '[' {
			return keyOther, nil
		}
		switch seq[1] {
		case 'A':
			return keyUp, nil
		case 'B':
			return keyDown, nil
		}
	}
	return keyOther, nil
}

func isTerminal(f *os.File) bool {
	fd := int(f.Fd())
	var state syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&state)))
	return errno == 0
}

func enableRawMode() (func(), error) {
	fd := int(os.Stdin.Fd())
	var oldState syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&oldState))); errno != 0 {
		return nil, fmt.Errorf("terminal not available")
	}

	newState := oldState
	newState.Lflag &^= syscall.ECHO | syscall.ICANON
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&newState))); errno != 0 {
		return nil, fmt.Errorf("enable raw mode: %v", errno)
	}

	return func() {
		syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)))
	}, nil
}
