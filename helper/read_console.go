package helper

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// TestMode disables real API calls; each command prints a stub message instead.
var TestMode bool

// stdinLineReader reads lines from os.Stdin. A single bufio.Reader is reused so
// consecutive reads (e.g. setup prompts) do not discard prefetched input.
var stdinLineReader *bufio.Reader

func ResetStdinReader() {
	stdinLineReader = nil
}

func readStdinLine() (string, error) {
	if stdinLineReader == nil {
		stdinLineReader = bufio.NewReader(os.Stdin)
	}
	line, err := stdinLineReader.ReadString('\n')
	return strings.TrimSpace(line), err
}

func ReadStdinLine() (string, error) {
	return readStdinLine()
}

// ReadPassword prints prompt, disables terminal echo, reads a line, then
// restores echo. Falls back to plain stdin read if the terminal cannot be
// configured (e.g. piped input).
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())

	// Try to disable echo via termios. If stdin is not a real terminal this
	// syscall will fail and we fall through to a plain read.
	var oldState syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&oldState))); errno == 0 {

		newState := oldState
		newState.Lflag &^= syscall.ECHO
		newState.Lflag |= syscall.ICANON | syscall.ISIG
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&newState))); errno == 0 {

			defer func() {
				_, _, _ = syscall.Syscall(syscall.SYS_IOCTL,
					uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)))
				fmt.Println()
			}()
		}
	}

	return readStdinLine()
}
