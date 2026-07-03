package helper

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// WriteStdoutMaybePaged prints content to stdout, piping through $PAGER when stdout
// is a TTY and the line count exceeds the terminal height (same idea as git log).
func WriteStdoutMaybePaged(content string) error {
	if !IsTerminal(os.Stdout) {
		_, err := io.WriteString(os.Stdout, content)
		return err
	}

	lines := lineCount(content)
	height, ok := stdoutHeight()
	if !ok || lines <= height {
		_, err := io.WriteString(os.Stdout, content)
		return err
	}
	return pageContent(content)
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

func pageContent(content string) error {
	name, args := pagerCommand()
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		// if the pager command fails, write the content to stdout
		_, writeErr := io.WriteString(os.Stdout, content)
		return writeErr
	}

	// if the pager command fails oddly, write the content to stdout
	if err := cmd.Wait(); err != nil && !isBenignPagerError(err) {
		return fmt.Errorf("pager: %w", err)
	}
	return nil
}

func isBenignPagerError(err error) bool { // if harmless - ok to ignore

	if errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	if strings.Contains(strings.ToLower(err.Error()), "broken pipe") {
		return true
	}
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func pagerCommand() (string, []string) {
	if p := strings.TrimSpace(os.Getenv("PAGER")); p != "" {
		parts := strings.Fields(p)
		return parts[0], parts[1:]
	}
	return "less", []string{"-R"}
}
