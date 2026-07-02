package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func withIdleExit(t *testing.T, fn func()) (code int, exited bool) {
	t.Helper()

	oldExit := idleStopExit
	idleStopExit = func(c int) {
		code = c
		exited = true
		panic("idleTestExit")
	}
	t.Cleanup(func() { idleStopExit = oldExit })

	defer func() {
		if recover() == "idleTestExit" {
			return
		}
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	fn()
	return code, exited
}

func captureIdleStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = oldStderr
		_ = r.Close()
	})

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestIdleRouterRejectsExtraArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "show extra", args: []string{"show", "mybox", "extra"}},
		{name: "delete extra", args: []string{"delete", "mybox", "extra"}},
		{name: "set extra", args: []string{"set", "mybox", "30", "extra"}},
		{name: "update extra", args: []string{"update", "mybox", "30", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			stderr := captureIdleStderr(t, func() {
				code, exited := withIdleExit(t, func() { IdleRouter(tt.args) })
				if !exited || code != 1 {
					t.Fatalf("IdleRouter() exit = %v code = %d, want exit 1", exited, code)
				}
			})
			if !strings.Contains(stderr, "usage:") {
				t.Fatalf("stderr = %q, want usage message", stderr)
			}
		})
	}
}
