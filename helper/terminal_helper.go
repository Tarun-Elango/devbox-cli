package helper

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func IsTerminal(f *os.File) bool {
	fd := int(f.Fd())
	var state syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&state)))
	return errno == 0
}

type KeyCode int

const (
	KeyUp KeyCode = iota
	KeyDown
	KeyEnter
	KeyCtrlC
	KeyOther
)

func ReadKey() (KeyCode, error) {
	var b [1]byte
	if _, err := os.Stdin.Read(b[:]); err != nil {
		return KeyOther, err
	}

	switch b[0] {
	case 3: // Ctrl+C
		return KeyCtrlC, nil
	case '\r', '\n':
		return KeyEnter, nil
	case 27: // ESC — arrow keys
		seq := make([]byte, 2)
		if _, err := os.Stdin.Read(seq[:]); err != nil {
			return KeyOther, err
		}
		if seq[0] != '[' {
			return KeyOther, nil
		}
		switch seq[1] {
		case 'A':
			return KeyUp, nil
		case 'B':
			return KeyDown, nil
		}
	}
	return KeyOther, nil
}

func EnableRawMode() (func(), error) {
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
		_, _, _ = syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)))
	}, nil
}
