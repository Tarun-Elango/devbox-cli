package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"outpost-cli/internal/sqliteutil"
)

const (
	outpostDir        = ".outpost"
	backupDirName     = ".outpost-backup"
	configFile        = "config.json"
	dbFile            = "outpost.db"
	backupInterval    = 24 * time.Hour
	timestampLayout   = "20060102-150405"
	timestampLayoutNS = "20060102-150405.000000000"
)

var backupMu sync.Mutex

// MaybeDaily creates a backup if at least 24 hours have passed since the last one.
// Best-effort: errors are ignored; backup blocks so short-lived CLI commands do not exit mid-backup.
func MaybeDaily() {

	dir, err := backupDir()
	if err != nil {
		return
	}
	if last, ok := latestBackupTime(dir); ok && time.Since(last) < backupInterval {
		return
	}
	runLocked(dir)
}

// BeforeConfigSave copies the current config and db before persisting changes in local mode.
// Blocks until the backup finishes so the snapshot is the previous version, not a race.
// Best-effort: errors are ignored.
func BeforeConfigSave() {
	dir, err := backupDir()
	if err != nil {
		return
	}
	runLocked(dir)
}

func runLocked(dir string) {
	backupMu.Lock()
	defer backupMu.Unlock()
	_ = create(dir)
}

// get the path to the backup directory
func backupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, backupDirName), nil
}

func outpostPaths() (configPath, dbPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	base := filepath.Join(home, outpostDir)
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
		t, err := parseBackupDirTime(e.Name())
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

// name to actual timestamp time.Time
func parseBackupDirTime(name string) (time.Time, error) {
	if t, err := time.Parse(timestampLayoutNS, name); err == nil {
		return t, nil
	}
	return time.Parse(timestampLayout, name)
}

func create(backupRoot string) error {
	configPath, dbPath, err := outpostPaths()
	if err != nil {
		return err
	}

	configExists := fileExists(configPath)
	dbExists := fileExists(dbPath)
	if !configExists && !dbExists {
		return nil
	}

	dest := filepath.Join(backupRoot, time.Now().UTC().Format(timestampLayoutNS))
	if err := os.MkdirAll(dest, 0700); err != nil {
		return err
	}

	if configExists {
		if err := copyFile(configPath, filepath.Join(dest, configFile)); err != nil {
			return err
		}
	}
	if dbExists {
		if err := backupDB(dbPath, filepath.Join(dest, dbFile)); err != nil {
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

// backupDB prefers VACUUM INTO for a consistent snapshot; falls back to a hot
// file copy when another process holds the database open.
func backupDB(srcPath, dstPath string) error {
	err := sqliteutil.WithRetry(func() error {
		return vacuumDB(srcPath, dstPath)
	})
	if err == nil {
		return nil
	}
	if !sqliteutil.IsBusy(err) {
		return err
	}

	// could not vacuum, so we need to copy the database file
	return hotCopyDB(srcPath, dstPath)
}

func vacuumDB(srcPath, dstPath string) error {
	conn, err := sqliteutil.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	absDst, err := filepath.Abs(dstPath)
	if err != nil {
		return err
	}
	escaped := strings.ReplaceAll(absDst, "'", "''")
	_, err = conn.Exec(fmt.Sprintf("VACUUM INTO '%s'", escaped))
	return err
}

/*
When checkpoint succeeds (“good”):
WAL contents are merged into the main .db and the WAL is truncated. Then only the main database file is copied to dstPath.

When checkpoint fails because the DB is busy (“not good” in the busy sense):
It falls back to copyDBFiles: the main .db plus any -wal and -shm files that exist, so the backup stays consistent even though the WAL wasn’t fully merged.
*/
func hotCopyDB(srcPath, dstPath string) error {
	conn, err := sqliteutil.Open(srcPath) // open the database connection
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	var busy int
	checkpointErr := sqliteutil.WithRetry(func() error {
		var logFrames, checkpointed int
		return conn.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed)
	})
	if checkpointErr != nil && !sqliteutil.IsBusy(checkpointErr) {
		return checkpointErr
	}
	if checkpointErr == nil && busy == 0 {
		// if the checkpoint succeeded and the busy flag is 0, copy the database file
		return copyFile(srcPath, dstPath)
	}
	return copyDBFiles(srcPath, dstPath) // if the checkpoint failed, copy the database file
}

func copyDBFiles(srcPath, dstPath string) error {
	if err := copyFile(srcPath, dstPath); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		sidecar := srcPath + suffix
		if !fileExists(sidecar) {
			continue
		}
		if err := copyFile(sidecar, dstPath+suffix); err != nil {
			return err
		}
	}
	return nil
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
