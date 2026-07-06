package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devbox-cli/helper"
	"devbox-cli/internal/config"
	"devbox-cli/service"
)

const testExit = "setupTestExit"

// pretend user types input
func withSetupStdin(t *testing.T, input string) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if input != "" {
		if _, err := io.WriteString(w, input); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	os.Stdin = r
	helper.ResetStdinReader()
	t.Cleanup(func() {
		os.Stdin = oldStdin
		helper.ResetStdinReader()
		_ = r.Close()
	})
}

// records what is written to stderr
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

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

// intercepts os.Exit calls
func withSetupExit(t *testing.T, fn func()) (code int, exited bool) {
	t.Helper()

	old := setupExit
	oldCommandParseExit := helper.CommandParseExit
	exitFn := func(c int) {
		code = c
		exited = true
		panic(testExit)
	}
	setupExit = exitFn
	helper.CommandParseExit = exitFn
	t.Cleanup(func() {
		setupExit = old
		helper.CommandParseExit = oldCommandParseExit
	})

	defer func() {
		if r := recover(); r != nil && r != testExit {
			panic(r)
		}
	}()

	fn()
	return code, exited
}

// use temp dir for test home
func withTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

// load the test config
func loadTestConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func TestSetup(t *testing.T) {
	regions := service.AllRegions() // get all regions

	t.Run("rejects extra args", func(t *testing.T) {
		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { Setup([]string{"extra"}) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "usage: devbox setup") {
			t.Fatalf("stderr = %q, want usage message", stderr)
		}
	})

	t.Run("access_key_read_error", func(t *testing.T) {
		withSetupStdin(t, "")

		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "error reading access key:") { // should have error
			t.Fatalf("stderr = %q, want access key read error", stderr)
		}
	})

	t.Run("intro_prompt_no_existing_credentials", func(t *testing.T) {
		withTestHome(t)
		withSetupStdin(t, "\n")

		out := captureStdout(t, func() {
			_, _ = withSetupExit(t, func() { Setup(nil) })
		})
		if !strings.Contains(out, "Enter access key and secret") {
			t.Fatalf("stdout = %q, want intro for new credentials", out)
		}
		if strings.Contains(out, "keep existing values") {
			t.Fatalf("stdout = %q, should not mention keeping existing values", out)
		}
	})

	t.Run("intro_prompt_with_existing_credentials", func(t *testing.T) {
		withTestHome(t)
		if err := service.SaveAWSCredentials("existing-secret", "existing-access", "us-east-1"); err != nil {
			t.Fatalf("seed config: %v", err)
		}
		withSetupStdin(t, "\n\n1\n")

		out := captureStdout(t, func() {
			_, _ = withSetupExit(t, func() { Setup(nil) })
		})
		if !strings.Contains(out, "keep existing values") {
			t.Fatalf("stdout = %q, want intro for keeping existing credentials", out)
		}
	})

	t.Run("empty_access_key", func(t *testing.T) {
		withTestHome(t)
		withSetupStdin(t, "\n")

		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "setup failed: access key is required") { // should have error
			t.Fatalf("stderr = %q, want access key required error", stderr)
		}
	})

	t.Run("secret_read_error", func(t *testing.T) {
		withSetupStdin(t, "access-key\n")

		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "error reading secret:") { // should have error
			t.Fatalf("stderr = %q, want secret read error", stderr)
		}
	})

	t.Run("empty_secret", func(t *testing.T) {
		withTestHome(t)
		withSetupStdin(t, "access-key\n\n")

		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "setup failed: secret is required") { // should have error
			t.Fatalf("stderr = %q, want secret required error", stderr)
		}
	})

	t.Run("invalid_region", func(t *testing.T) {
		withSetupStdin(t, "access-key\nsecret-key\nnot-a-region\n") // input invalid region

		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "error selecting region:") { // should have error
			t.Fatalf("stderr = %q, want region selection error", stderr)
		}
	})

	t.Run("save_error", func(t *testing.T) {
		home := t.TempDir()
		// create a file .devbox, so it messes up the config save
		if err := os.WriteFile(filepath.Join(home, ".devbox"), []byte("block"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOME", home)
		withSetupStdin(t, "access-key\nsecret-key\n1\n")

		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "load config:") && !strings.Contains(stderr, "save config:") {
			t.Fatalf("stderr = %q, want load or save config error", stderr)
		}
	})

	t.Run("success_by_region_number", func(t *testing.T) {
		withTestHome(t)
		withSetupStdin(t, "my-access\nmy-secret\n1\n")

		out := captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})

		cfg := loadTestConfig(t)
		if cfg.AwsSecret != "my-secret" || cfg.AwsAccessKey != "my-access" || cfg.AwsRegion != regions[0].ID {
			t.Fatalf("config = %+v, want secret/access/region saved", cfg)
		}
		if !strings.Contains(out, "Credentials saved") {
			t.Fatalf("stdout = %q, want save confirmation", out)
		}
	})

	t.Run("success_by_region_id", func(t *testing.T) {
		withTestHome(t)
		withSetupStdin(t, "other-access\nother-secret\nus-west-2\n")

		captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})

		cfg := loadTestConfig(t)
		if cfg.AwsSecret != "other-secret" || cfg.AwsAccessKey != "other-access" || cfg.AwsRegion != "us-west-2" {
			t.Fatalf("config = %+v, want credentials saved with us-west-2", cfg)
		}
	})

	t.Run("keeps_existing_credentials_on_empty_input", func(t *testing.T) {
		withTestHome(t)
		if err := service.SaveAWSCredentials("existing-secret", "existing-access", "us-east-1"); err != nil {
			t.Fatalf("seed config: %v", err)
		}
		withSetupStdin(t, "\n\n1\n")

		captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})

		cfg := loadTestConfig(t)
		if cfg.AwsSecret != "existing-secret" || cfg.AwsAccessKey != "existing-access" {
			t.Fatalf("config = %+v, want existing credentials preserved", cfg)
		}
	})

	t.Run("updates_one_credential_keeps_other", func(t *testing.T) {
		withTestHome(t)
		if err := service.SaveAWSCredentials("existing-secret", "existing-access", "us-east-1"); err != nil {
			t.Fatalf("seed config: %v", err)
		}
		withSetupStdin(t, "new-access\n\n1\n")

		captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { Setup(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})

		cfg := loadTestConfig(t)
		if cfg.AwsAccessKey != "new-access" || cfg.AwsSecret != "existing-secret" {
			t.Fatalf("config = %+v, want updated access key with preserved secret", cfg)
		}
	})
}

func TestSelectRegionFallback(t *testing.T) {
	regions := service.AllRegions()

	t.Run("select_by_number", func(t *testing.T) {
		withSetupStdin(t, "2\n")
		got, err := selectRegionFallback(regions)
		if err != nil {
			t.Fatalf("selectRegionFallback: %v", err)
		}
		if got != regions[1].ID {
			t.Fatalf("got %q, want %q", got, regions[1].ID)
		}
	})

	t.Run("select_by_id", func(t *testing.T) {
		withSetupStdin(t, "eu-west-1\n")
		got, err := selectRegionFallback(regions)
		if err != nil {
			t.Fatalf("selectRegionFallback: %v", err)
		}
		if got != "eu-west-1" {
			t.Fatalf("got %q, want eu-west-1", got)
		}
	})

	t.Run("number_out_of_range", func(t *testing.T) {
		withSetupStdin(t, "999\n")
		_, err := selectRegionFallback(regions)
		if err == nil || !strings.Contains(err.Error(), `invalid region "999"`) {
			t.Fatalf("got %v, want invalid region error", err)
		}
	})

	t.Run("number_zero", func(t *testing.T) {
		withSetupStdin(t, "0\n")
		_, err := selectRegionFallback(regions)
		if err == nil {
			t.Fatal("expected error for region number 0")
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		withSetupStdin(t, "\n")
		_, err := selectRegionFallback(regions)
		if err == nil || !strings.Contains(err.Error(), `invalid region ""`) {
			t.Fatalf("got %v, want invalid empty region error", err)
		}
	})

	t.Run("read_error", func(t *testing.T) {
		withSetupStdin(t, "")
		_, err := selectRegionFallback(regions)
		if err == nil {
			t.Fatal("expected read error")
		}
	})
}

func TestClearCreds(t *testing.T) {

	t.Run("extra_args", func(t *testing.T) {
		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds([]string{"extra"}) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "usage: devbox clear-creds") {
			t.Fatalf("stderr = %q, want usage message", stderr)
		}
	})

	t.Run("declined_empty", func(t *testing.T) {
		withSetupStdin(t, "\n")

		out := captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})
		if !strings.Contains(out, "Aborted.") {
			t.Fatalf("stdout = %q, want Aborted message", out)
		}
	})

	t.Run("declined_n", func(t *testing.T) {
		withSetupStdin(t, "n\n")

		out := captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})
		if !strings.Contains(out, "Aborted.") {
			t.Fatalf("stdout = %q, want Aborted message", out)
		}
	})

	t.Run("declined_N", func(t *testing.T) {
		withSetupStdin(t, "N\n")

		out := captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})
		if !strings.Contains(out, "Aborted.") {
			t.Fatalf("stdout = %q, want Aborted message", out)
		}
	})

	t.Run("clear_error", func(t *testing.T) {
		home := t.TempDir()
		devboxDir := filepath.Join(home, ".devbox")
		if err := os.MkdirAll(devboxDir, 0700); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(devboxDir, "config.json")
		if err := os.WriteFile(cfgPath, []byte("{"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOME", home)
		withSetupStdin(t, "y\n")

		stderr := captureStderr(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds(nil) })
			if !exited || code != 1 {
				t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
			}
		})
		if !strings.Contains(stderr, "clear credentials:") {
			t.Fatalf("stderr = %q, want clear credentials error", stderr)
		}
	})

	t.Run("success_y", func(t *testing.T) {
		withTestHome(t)
		if err := service.SaveAWSCredentials("secret", "access", "us-east-1"); err != nil {
			t.Fatalf("seed config: %v", err)
		}
		withSetupStdin(t, "y\n")

		out := captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})

		cfg := loadTestConfig(t)
		if cfg.AwsSecret != "" || cfg.AwsAccessKey != "" {
			t.Fatalf("config = %+v, want cleared credentials", cfg)
		}
		if !strings.Contains(out, "AWS credentials cleared") {
			t.Fatalf("stdout = %q, want clear confirmation", out)
		}
	})

	t.Run("success_Y", func(t *testing.T) {
		withTestHome(t)
		if err := service.SaveAWSCredentials("secret2", "access2", "us-west-2"); err != nil {
			t.Fatalf("seed config: %v", err)
		}
		withSetupStdin(t, "Y\n")

		captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})

		cfg := loadTestConfig(t)
		if cfg.AwsSecret != "" || cfg.AwsAccessKey != "" {
			t.Fatalf("config = %+v, want cleared credentials", cfg)
		}
	})

	t.Run("preserves_other_config_fields", func(t *testing.T) {
		withTestHome(t)
		if err := service.SaveAWSCredentials("secret", "access", "us-east-1"); err != nil {
			t.Fatalf("seed config: %v", err)
		}

		cfgPath, err := config.ConfigPath()
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatal(err)
		}
		raw["mode"] = json.RawMessage(`"local"`)
		updated, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cfgPath, updated, 0600); err != nil {
			t.Fatal(err)
		}

		withSetupStdin(t, "y\n")
		captureStdout(t, func() {
			code, exited := withSetupExit(t, func() { ClearCreds(nil) })
			if exited {
				t.Fatalf("unexpected exit %d", code)
			}
		})

		cfg := loadTestConfig(t)
		if cfg.Mode != "local" {
			t.Fatalf("Mode = %q, want preserved mode field", cfg.Mode)
		}
		if cfg.AwsSecret != "" || cfg.AwsAccessKey != "" {
			t.Fatalf("config = %+v, want only credentials cleared", cfg)
		}
	})
}
