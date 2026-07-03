package helper

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pager: %w", err)
	}
	return nil
}

func pagerCommand() (string, []string) {
	if p := strings.TrimSpace(os.Getenv("PAGER")); p != "" {
		parts := strings.Fields(p)
		return parts[0], parts[1:]
	}
	return "less", []string{"-R"}
}
