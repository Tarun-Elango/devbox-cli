//go:build !windows

package helper

import (
	"os"
	"syscall"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func stdoutHeight() (int, bool) {
	ws, ok := stdoutSize()
	if !ok {
		return 0, false
	}
	return ws.Row, true
}

// StdoutWidth returns the current terminal width, defaulting to 80 columns
// when stdout is not attached to a terminal or its size cannot be determined.
func StdoutWidth() int {
	ws, ok := stdoutSize()
	if !ok {
		return 80
	}
	return ws.Col
}

type terminalSize struct {
	Row int
	Col int
}

func stdoutSize() (terminalSize, bool) {
	fd := int(os.Stdout.Fd())
	var ws winsize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 || ws.Row == 0 || ws.Col == 0 {
		return terminalSize{}, false
	}
	return terminalSize{Row: int(ws.Row), Col: int(ws.Col)}, true
}
