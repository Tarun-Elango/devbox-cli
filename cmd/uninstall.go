package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"outpost-cli/helper"
)

const (
	outpostDataDir    = ".outpost"
	outpostBackupDir  = ".outpost-backup"
	outpostPathMarker = "# outpost"
)

var (
	userHomeDirFn = os.UserHomeDir
)

// Uninstall removes the outpost binary, local data directories, and PATH entries
// added by scripts/install.sh.
func Uninstall(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost uninstall")

	fmt.Println("This will remove outpost, ~/.outpost, ~/.outpost-backup, and PATH entries added by install.")
	fmt.Print("Uninstall outpost? [y/N] ")
	answer, err := helper.ReadStdinLine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read uninstall confirmation: %v\n", err)
		setupExit(1)
		return
	}
	if !isYes(answer) {
		fmt.Println("Uninstall skipped.")
		return
	}

	exe, err := osExecutableFn() // get the path to the current outpost binary
	if err != nil {
		fmt.Fprintf(os.Stderr, "locate current outpost binary: %v\n", err)
		setupExit(1)
		return
	}
	installDir := filepath.Dir(exe) // get the directory of the current outpost binary

	home, err := userHomeDirFn() // get the home directory of the current user
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve home directory: %v\n", err)
		setupExit(1)
		return
	}

	// delete the ~/.outpost directory
	if err := removeDataDir(filepath.Join(home, outpostDataDir)); err != nil {
		fmt.Fprintf(os.Stderr, "remove ~/.outpost: %v\n", err)
		setupExit(1)
		return
	}
	fmt.Println("Removed ~/.outpost")

	// delete the ~/.outpost-backup directory if it exists
	backupDir := filepath.Join(home, outpostBackupDir)
	if fileExists(backupDir) {
		if err := removeDataDir(backupDir); err != nil {
			fmt.Fprintf(os.Stderr, "remove ~/.outpost-backup: %v\n", err)
			setupExit(1)
			return
		}
		fmt.Println("Removed ~/.outpost-backup")
	}

	// clear the PATH entries from the shell config
	updated, err := clearoutpostPath(home, installDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "clear PATH entries: %v\n", err)
		setupExit(1)
		return
	}
	for _, path := range updated {
		fmt.Printf("Removed outpost PATH entry from %s\n", path)
	}
	if len(updated) == 0 {
		fmt.Println("No outpost PATH entries found in shell config.")
	}

	// delete the outpost binary
	if err := os.Remove(exe); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "outpost binary not found at %s\n", exe)
		} else {
			fmt.Fprintf(os.Stderr, "remove outpost binary: %v\n", err)
		}
		setupExit(1)
		return
	}
	fmt.Printf("Removed %s\n", exe)

	// delete the install directory if it is empty
	if empty, err := dirIsEmpty(installDir); err == nil && empty {
		if err := os.Remove(installDir); err == nil {
			fmt.Printf("Removed empty directory %s\n", installDir)
		}
	}

	fmt.Println("outpost uninstalled. Restart your shell if PATH was updated.")
}

func removeDataDir(path string) error {
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func clearoutpostPath(home, installDir string) ([]string, error) {
	var updated []string
	for _, rcPath := range shellRCFiles(home) {
		changed, err := cleanShellRCFile(rcPath, installDir, home)
		if err != nil {
			return nil, err
		}
		if changed {
			updated = append(updated, rcPath)
		}
	}
	return updated, nil
}

func shellRCFiles(home string) []string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = defaultShellForOS()
	}

	primary := primaryShellRC(home, shell)
	candidates := []string{
		primary,
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}

	seen := make(map[string]struct{}, len(candidates))
	var out []string
	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		if _, err := os.Stat(path); err == nil {
			out = append(out, path)
		}
	}
	return out
}

func defaultShellForOS() string {
	if runtime.GOOS == "darwin" {
		return "/bin/zsh"
	}
	return "/bin/bash"
}

func primaryShellRC(home, shell string) string {
	switch {
	case strings.HasSuffix(shell, "zsh"):
		return filepath.Join(home, ".zshrc")
	case strings.HasSuffix(shell, "bash"):
		if runtime.GOOS == "darwin" {
			bashProfile := filepath.Join(home, ".bash_profile")
			bashrc := filepath.Join(home, ".bashrc")
			if fileExists(bashProfile) || !fileExists(bashrc) {
				return bashProfile
			}
			return bashrc
		}
		return filepath.Join(home, ".bashrc")
	default:
		for _, candidate := range []string{
			filepath.Join(home, ".zshrc"),
			filepath.Join(home, ".bash_profile"),
			filepath.Join(home, ".bashrc"),
			filepath.Join(home, ".profile"),
		} {
			if fileExists(candidate) {
				return candidate
			}
		}
		return filepath.Join(home, ".profile")
	}
}

func cleanShellRCFile(path, installDir, home string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	cleaned := cleanShellRCContent(string(content), installDir, home)
	if cleaned == string(content) {
		return false, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(cleaned), info.Mode().Perm()); err != nil {
		return false, err
	}
	return true, nil
}

// cleanShellRCContent removes only the PATH block that install.sh itself
// wrote: the "# outpost" marker comment together with the PATH export line
// immediately following it. Lines are never matched by content alone (e.g.
// any line that happens to mention ~/.local/bin), since that would risk
// deleting unrelated PATH entries added by other tools (pipx, cargo, npm,
// etc.) that share the same directory.
func cleanShellRCContent(content, installDir, home string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	changed := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == outpostPathMarker && i+1 < len(lines) && isoutpostPathExport(lines[i+1], installDir, home) {
			changed = true
			i++ // also skip the PATH export line right after the marker
			continue
		}
		out = append(out, line)
	}

	if !changed {
		return content
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

func isoutpostPathExport(line, installDir, home string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "PATH") {
		return false
	}
	if strings.Contains(line, installDir) {
		return true
	}

	defaultInstall := filepath.Join(home, ".local", "bin")
	if installDir != defaultInstall {
		return false
	}
	return strings.Contains(line, "$HOME/.local/bin") ||
		strings.Contains(line, "${HOME}/.local/bin")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirIsEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}
