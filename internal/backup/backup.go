package backup

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	devboxDir       = ".devbox"
	backupDirName   = ".devbox-backup"
	configFile      = "config.json"
	dbFile          = "devbox.db"
	backupInterval  = 24 * time.Hour
	timestampLayout = "20060102-150405"
)

var backupMu sync.Mutex

// MaybeDaily creates a backup if at least 24 hours have passed since the last one.
// Best-effort: errors are ignored; backup runs in the background.
func MaybeDaily() {
	if !isLocalMode() {
		return
	}
	dir, err := backupDir()
	if err != nil {
		return
	}
	if last, ok := latestBackupTime(dir); ok && time.Since(last) < backupInterval {
		return
	}
	runAsync(dir)
}

// BeforeConfigSave creates a backup before persisting config changes in local mode.
// Best-effort: errors are ignored; backup runs in the background.
func BeforeConfigSave(mode string) {
	if mode != "" && mode != "local" {
		return
	}
	dir, err := backupDir()
	if err != nil {
		return
	}
	runAsync(dir)
}

func runAsync(dir string) {
	go func() {
		backupMu.Lock()
		defer backupMu.Unlock()
		_ = create(dir)
	}()
}

func isLocalMode() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(filepath.Join(home, devboxDir, configFile))
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		return false
	}
	var cfg struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	return cfg.Mode == "" || cfg.Mode == "local"
}

// get the path to the backup directory
func backupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, backupDirName), nil
}

func devboxPaths() (configPath, dbPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	base := filepath.Join(home, devboxDir)
	return filepath.Join(base, configFile), filepath.Join(base, dbFile), nil
}

func latestBackupTime(dir string) (time.Time, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, false
	}
	var latest time.Time
	var found bool
	// for each entry in the backup directory,
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, err := time.Parse(timestampLayout, e.Name())
		if err != nil {
			continue
		}
		if !found || t.After(latest) {
			latest = t
			found = true
		}
	}
	return latest, found
}

func create(backupRoot string) error {
	configPath, dbPath, err := devboxPaths()
	if err != nil {
		return err
	}

	configExists := fileExists(configPath)
	dbExists := fileExists(dbPath)
	if !configExists && !dbExists {
		return nil
	}

	dest := filepath.Join(backupRoot, time.Now().Format(timestampLayout))
	if err := os.MkdirAll(dest, 0700); err != nil {
		return err
	}

	if configExists {
		if err := copyFile(configPath, filepath.Join(dest, configFile)); err != nil {
			return err
		}
	}
	if dbExists {
		if err := copyFile(dbPath, filepath.Join(dest, dbFile)); err != nil {
			return err
		}
	}
	// only remove the backups if copying was successful
	if err := removeOldBackupsExcept(backupRoot, filepath.Base(dest)); err != nil {
		return err
	}
	return nil
}

func removeOldBackupsExcept(dir, keep string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {

		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// loop all the entries
	for _, e := range entries {
		// only remove non-directory entries and the keep entry
		if !e.IsDir() || e.Name() == keep {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
