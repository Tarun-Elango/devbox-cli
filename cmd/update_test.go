package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func withUpdateStubs(t *testing.T) {
	t.Helper()

	oldFetch := fetchLatestVersionFn
	oldInstall := installLatestFn
	oldCurrent := currentVersionFn
	oldExecutable := osExecutableFn

	t.Cleanup(func() {
		fetchLatestVersionFn = oldFetch
		installLatestFn = oldInstall
		currentVersionFn = oldCurrent
		osExecutableFn = oldExecutable
	})
}

func TestLatestVersionFromTags(t *testing.T) {
	got, err := latestVersionFromTags([]githubTag{
		{Name: "latest"},
		{Name: "v0.9.0"},
		{Name: "v0.10.0"},
		{Name: "not-a-version"},
		{Name: "v0.4.0"},
	})
	if err != nil {
		t.Fatalf("latestVersionFromTags: %v", err)
	}
	if got != "0.10.0" {
		t.Fatalf("got %q, want 0.10.0", got)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{a: "0.4.0", b: "v0.4.0", want: 0},
		{a: "0.10.0", b: "0.9.9", want: 1},
		{a: "1.0.0", b: "2.0.0", want: -1},
	}

	for _, tt := range tests {
		got, err := compareVersions(tt.a, tt.b)
		if err != nil {
			t.Fatalf("compareVersions(%q, %q): %v", tt.a, tt.b, err)
		}
		if got != tt.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestUpdateRejectsExtraArgs(t *testing.T) {
	stderr := captureStderr(t, func() {
		code, exited := withSetupExit(t, func() { Update([]string{"extra"}) })
		if !exited || code != 1 {
			t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
		}
	})
	if !strings.Contains(stderr, "usage: devbox update") {
		t.Fatalf("stderr = %q, want usage message", stderr)
	}
}

func TestUpdateAlreadyCurrent(t *testing.T) {
	withUpdateStubs(t)
	currentVersionFn = func() string { return "0.4.0" }
	fetchLatestVersionFn = func(context.Context) (string, error) { return "0.4.0", nil }

	out := captureStdout(t, func() {
		code, exited := withSetupExit(t, func() { Update(nil) })
		if exited {
			t.Fatalf("unexpected exit %d", code)
		}
	})
	if !strings.Contains(out, "devbox 0.4.0 is up to date.") {
		t.Fatalf("stdout = %q, want up-to-date message", out)
	}
}

func TestUpdateDeclined(t *testing.T) {
	withUpdateStubs(t)
	withSetupStdin(t, "n\n")
	currentVersionFn = func() string { return "0.4.0" }
	fetchLatestVersionFn = func(context.Context) (string, error) { return "0.5.0", nil }
	installLatestFn = func(context.Context, string) error {
		t.Fatal("installLatestFn should not be called")
		return nil
	}

	out := captureStdout(t, func() {
		code, exited := withSetupExit(t, func() { Update(nil) })
		if exited {
			t.Fatalf("unexpected exit %d", code)
		}
	})
	if !strings.Contains(out, "Update skipped.") {
		t.Fatalf("stdout = %q, want skipped message", out)
	}
}

func TestUpdateAcceptedInstallsToExecutableDir(t *testing.T) {
	withUpdateStubs(t)
	withSetupStdin(t, "yes\n")
	currentVersionFn = func() string { return "0.4.0" }
	fetchLatestVersionFn = func(context.Context) (string, error) { return "0.5.0", nil }
	osExecutableFn = func() (string, error) { return "/tmp/devbox-test/bin/devbox", nil }

	var gotInstallDir string
	installLatestFn = func(_ context.Context, installDir string) error {
		gotInstallDir = installDir
		return nil
	}

	out := captureStdout(t, func() {
		code, exited := withSetupExit(t, func() { Update(nil) })
		if exited {
			t.Fatalf("unexpected exit %d", code)
		}
	})
	if gotInstallDir != "/tmp/devbox-test/bin" {
		t.Fatalf("install dir = %q, want /tmp/devbox-test/bin", gotInstallDir)
	}
	if !strings.Contains(out, "devbox 0.5.0 is available. You have 0.4.0.") {
		t.Fatalf("stdout = %q, want available message", out)
	}
}

func TestUpdateFetchErrorExits(t *testing.T) {
	withUpdateStubs(t)
	currentVersionFn = func() string { return "0.4.0" }
	fetchLatestVersionFn = func(context.Context) (string, error) {
		return "", errors.New("network down")
	}

	stderr := captureStderr(t, func() {
		code, exited := withSetupExit(t, func() { Update(nil) })
		if !exited || code != 1 {
			t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
		}
	})
	if !strings.Contains(stderr, "check for update: network down") {
		t.Fatalf("stderr = %q, want fetch error", stderr)
	}
}
