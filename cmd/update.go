package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"outpost-cli/helper"
	"outpost-cli/internal/version"
)

const (
	updateTagsURL = "https://api.github.com/repos/Tarun-Elango/outpost/tags"
	// Pinned to the "latest" release tag (not the mutable main branch) so
	// the fetched script always matches a published release.
	updateInstallScript = "https://raw.githubusercontent.com/Tarun-Elango/outpost/latest/scripts/install.sh"
)

var (
	fetchLatestVersionFn = fetchLatestVersion
	installLatestFn      = installLatest
	currentVersionFn     = version.String
	osExecutableFn       = os.Executable
)

type githubTag struct {
	Name string `json:"name"`
}

func Update(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost update")

	current := currentVersionFn()
	latest, err := fetchLatestVersionFn(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "check for update: %v\n", err)
		setupExit(1)
		return
	}

	cmp, err := compareVersions(current, latest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compare versions: %v\n", err)
		setupExit(1)
		return
	}
	if cmp >= 0 {
		fmt.Printf("outpost %s is up to date.\n", current)
		return
	}

	fmt.Printf("outpost %s is available. You have %s.\n", latest, current)
	fmt.Print("Update now? [y/N] ")
	answer, err := helper.ReadStdinLine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read update confirmation: %v\n", err)
		setupExit(1)
		return
	}
	if !isYes(answer) {
		fmt.Println("Update skipped.")
		return
	}

	exe, err := osExecutableFn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "locate current outpost binary: %v\n", err)
		setupExit(1)
		return
	}

	installDir := filepath.Dir(exe)
	if err := installLatestFn(context.Background(), installDir); err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
		setupExit(1)
		return
	}
}

func fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateTagsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "outpost-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub tags returned %s", resp.Status)
	}

	var tags []githubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}
	return latestVersionFromTags(tags)
}

func latestVersionFromTags(tags []githubTag) (string, error) {
	latest := ""
	for _, tag := range tags {
		candidate := normalizeVersion(tag.Name)
		if candidate == "" {
			continue
		}
		if latest == "" {
			latest = candidate
			continue
		}
		cmp, err := compareVersions(candidate, latest)
		if err != nil {
			continue
		}
		if cmp > 0 {
			latest = candidate
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no version tags found")
	}
	return latest, nil
}

func compareVersions(a, b string) (int, error) {
	aParts, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	bParts, err := parseVersion(b)
	if err != nil {
		return 0, err
	}

	for i := range aParts {
		if aParts[i] > bParts[i] {
			return 1, nil
		}
		if aParts[i] < bParts[i] {
			return -1, nil
		}
	}
	return 0, nil
}

func parseVersion(value string) ([3]int, error) {
	var out [3]int
	value = normalizeVersion(value)
	if value == "" {
		return out, fmt.Errorf("invalid version %q", value)
	}

	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("invalid version %q", value)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, fmt.Errorf("invalid version %q", value)
		}
		out[i] = n
	}
	return out, nil
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	if value == "latest" {
		return ""
	}
	return value
}

func isYes(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "y" || value == "yes"
}

func installLatest(ctx context.Context, installDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateInstallScript, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("install script returned %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "outpost-install-*.sh")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	execCmd := exec.CommandContext(ctx, "bash", tmp.Name())
	execCmd.Env = append(os.Environ(), "INSTALL_DIR="+installDir)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	return execCmd.Run()
}
