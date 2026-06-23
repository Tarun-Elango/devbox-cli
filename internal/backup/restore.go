package backup

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// RestoreConfigIfNeeded copies config.json from the latest backup when the live
// file is missing or unreadable. Best-effort: errors are ignored.
func RestoreConfigIfNeeded() {
	path, err := configPath()
	if err != nil {
		return
	}
	if isConfigValid(path) {
		return
	}
	restoreFile(path, configFile)
}

// RestoreDBIfNeeded copies devbox.db from the latest backup when the live file
// is missing or unusable. Best-effort: errors are ignored.
func RestoreDBIfNeeded() {
	path, err := dbPath()
	if err != nil {
		return
	}
	if isDBValid(path) {
		return
	}
	restoreFile(path, dbFile)
}

func configPath() (string, error) {
	configPath, _, err := devboxPaths()
	return configPath, err
}

func dbPath() (string, error) {
	_, dbPath, err := devboxPaths()
	return dbPath, err
}

func isConfigValid(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var cfg map[string]json.RawMessage
	return json.Unmarshal(data, &cfg) == nil
}

func isDBValid(path string) bool {
	if !fileExists(path) {
		return false
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()
	conn.SetMaxOpenConns(1)
	var result string

	// we do a PRAGMA quick_check to check if the database is valid
	if err := conn.QueryRow("PRAGMA quick_check").Scan(&result); err != nil {
		return false
	}
	return result == "ok"
}

// this checks the edge case where we have two backups one incomplete and 1 complete
// and we make sure to check the file which has the file that we want (name)
func latestBackupDirWithFile(name string) (string, bool) {
	dir, err := backupDir()
	if err != nil {
		return "", false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	var newest time.Time
	var result string
	var found bool
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// t -> is the timestamp of the backup directory
		t, err := parseBackupDirTime(e.Name())
		if err != nil {
			continue
		}
		if !fileExists(filepath.Join(dir, e.Name(), name)) {
			continue
		}
		if !found || t.After(newest) {
			newest = t
			result = filepath.Join(dir, e.Name())
			found = true
		}
	}
	return result, found
}

// restore file ( based on name - either config.json or devbox.db)
func restoreFile(destPath, name string) {
	backupRoot, ok := latestBackupDirWithFile(name)
	if !ok {
		return
	}
	src := filepath.Join(backupRoot, name)
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0700); err != nil { // create the destination directory
		return
	}

	tempFile, err := os.CreateTemp(destDir, filepath.Base(destPath)+".tmp-*")
	if err != nil {
		return
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()
	defer func() { _ = os.Remove(tempPath) }()

	if err := copyFile(src, tempPath); err != nil { // copy the file
		return
	}
	if err := os.Chmod(tempPath, 0600); err != nil { // set the file permissions before replacing the destination
		return
	}
	if err := os.Rename(tempPath, destPath); err != nil { // atomically replace the destination file
		return
	}
}
