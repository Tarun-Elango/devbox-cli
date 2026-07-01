package cmd

import (
	"strings"
	"testing"
)

func TestVersionRejectsExtraArgs(t *testing.T) {
	stderr := captureStderr(t, func() {
		code, exited := withSetupExit(t, func() { Version([]string{"xyz"}) })
		if !exited || code != 1 {
			t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
		}
	})
	if !strings.Contains(stderr, "usage: devbox version") {
		t.Fatalf("stderr = %q, want usage message", stderr)
	}
}
