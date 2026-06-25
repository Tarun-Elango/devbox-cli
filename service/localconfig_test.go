package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestSSHConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".ssh", "config")
	t.Setenv("HOME", filepath.Dir(filepath.Dir(path)))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}
	return path
}

func TestRenameHostPreservesOptionsAndOtherHosts(t *testing.T) {
	path := writeTestSSHConfig(t, "Host github.com\n    HostName github.com\n\nHost devbox-alpha alpha-extra\n    HostName 10.0.0.1\n    User ec2-user\n")

	if err := RenameHost("alpha", "beta"); err != nil {
		t.Fatalf("rename host: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "Host devbox-beta alpha-extra\n    HostName 10.0.0.1\n    User ec2-user") {
		t.Fatalf("renamed host block not preserved:\n%s", got)
	}
	if strings.Contains(got, "devbox-alpha") {
		t.Fatalf("old host still present:\n%s", got)
	}
}

func TestRenameHostRejectsExistingTarget(t *testing.T) {
	writeTestSSHConfig(t, "Host devbox-alpha\n    HostName 10.0.0.1\n\nHost devbox-beta\n    HostName 10.0.0.2\n")

	err := RenameHost("alpha", "beta")
	if err == nil {
		t.Fatal("expected duplicate host error")
	}
	if !strings.Contains(err.Error(), `host "devbox-beta" already exists`) {
		t.Fatalf("unexpected duplicate host error: %v", err)
	}
}

func TestUpdateSSHConfigWithRetryRetriesFailures(t *testing.T) {
	path := writeTestSSHConfig(t, "")
	attempts := 0

	err := updateSSHConfigWithRetry(func(content string) (string, error) {
		attempts++
		if attempts < sshConfigUpdateAttempts {
			return "", fmt.Errorf("temporary failure")
		}
		return "Host devbox-beta\n", nil
	})
	if err != nil {
		t.Fatalf("update ssh config with retry: %v", err)
	}
	if attempts != sshConfigUpdateAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, sshConfigUpdateAttempts)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	if string(data) != "Host devbox-beta\n" {
		t.Fatalf("ssh config = %q, want %q", string(data), "Host devbox-beta\n")
	}
}
