package helper

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func withStdin(t *testing.T, input string) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	os.Stdin = r
	ResetStdinReader()
	t.Cleanup(func() {
		os.Stdin = oldStdin
		ResetStdinReader()
		_ = r.Close()
	})

	go func() {
		if input != "" {
			_, _ = io.WriteString(w, input)
		}
		_ = w.Close()
	}()
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	done := make(chan string, 1)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestReadPassword_readsAndTrims(t *testing.T) {
	withStdin(t, "  secret123  \n")

	got, err := ReadPassword("Password: ")
	if err != nil {
		t.Fatalf("ReadPassword: %v", err)
	}
	if got != "secret123" {
		t.Fatalf("got %q, want %q", got, "secret123")
	}
}

func TestReadPassword_emptyLine(t *testing.T) {
	withStdin(t, "\n")

	got, err := ReadPassword("Password: ")
	if err != nil {
		t.Fatalf("ReadPassword: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}

func TestReadPassword_eof(t *testing.T) {
	withStdin(t, "")

	_, err := ReadPassword("Password: ")
	if err == nil {
		t.Fatal("expected EOF error")
	}
}

func TestReadPassword_printsPrompt(t *testing.T) {
	withStdin(t, "x\n")

	out := captureStdout(t, func() {
		if _, err := ReadPassword("Secret: "); err != nil {
			t.Fatalf("ReadPassword: %v", err)
		}
	})

	if !strings.HasPrefix(out, "Secret: ") {
		t.Fatalf("stdout %q does not start with prompt", out)
	}
}
