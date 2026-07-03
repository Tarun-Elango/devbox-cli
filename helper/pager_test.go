package helper

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestLineCount(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{input: "", want: 0},
		{input: "one", want: 1},
		{input: "one\n", want: 1},
		{input: "one\ntwo", want: 2},
		{input: "one\ntwo\n", want: 2},
		{input: "a\nb\nc\n", want: 3},
	}

	for _, tt := range tests {
		if got := lineCount(tt.input); got != tt.want {
			t.Fatalf("lineCount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestPagerCommandUsesEnv(t *testing.T) {
	t.Setenv("PAGER", "less -FRX")

	name, args := pagerCommand()
	if name != "less" || len(args) != 1 || args[0] != "-FRX" {
		t.Fatalf("pagerCommand() = (%q, %v), want (less, [-FRX])", name, args)
	}
}

func TestPagerCommandDefault(t *testing.T) {
	t.Setenv("PAGER", "")

	name, args := pagerCommand()
	if name != "less" || len(args) != 1 || args[0] != "-R" {
		t.Fatalf("pagerCommand() = (%q, %v), want (less, [-R])", name, args)
	}
}

func runAndCaptureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w

	outCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		var b strings.Builder
		_, copyErr := io.Copy(&b, r)
		_ = r.Close()
		outCh <- b.String()
		errCh <- copyErr
	}()

	fnErr := fn()
	_ = w.Close()
	os.Stdout = oldStdout

	if copyErr := <-errCh; copyErr != nil {
		t.Fatalf("io.Copy(stdout) error = %v", copyErr)
	}
	return <-outCh, fnErr
}

func TestPageContentFallsBackWhenPagerMissing(t *testing.T) {
	t.Setenv("PAGER", "/definitely/not/a/pager")

	content := "hello\nworld\n"
	output, err := runAndCaptureStdout(t, func() error {
		return pageContent(content)
	})
	if err != nil {
		t.Fatalf("pageContent() error = %v, want nil", err)
	}
	if output != content {
		t.Fatalf("pageContent() output = %q, want %q", output, content)
	}
}

func TestPageContentIgnoresBenignPagerExit(t *testing.T) {
	t.Setenv("PAGER", "false")

	content := "line\n"
	_, err := runAndCaptureStdout(t, func() error {
		return pageContent(content)
	})
	if err != nil {
		t.Fatalf("pageContent() error = %v, want nil", err)
	}
}
