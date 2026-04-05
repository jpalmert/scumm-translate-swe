// Package backup creates safety copies of files before they are overwritten.
package backup

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrBackupExists is returned by Create when a backup file already existed and
// was left untouched. The returned backup path is still valid.
var ErrBackupExists = errors.New("backup already existed from a previous run")

// Create copies src to src+".bak".
//
// If a backup already exists it is NOT overwritten — the first backup is the
// canonical original, and subsequent calls preserve it intact.
//
// Returns the backup path and any error. On success the caller may safely
// overwrite the original file.
func Create(path string) (string, error) {
	backupPath := path + ".bak"

	if _, err := os.Stat(backupPath); err == nil {
		// Backup already exists — keep the first backup as the canonical original.
		return backupPath, ErrBackupExists
	}

	src, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s for backup: %w", path, err)
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("create backup %s: %w", backupPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(backupPath) // clean up partial write
		return "", fmt.Errorf("copy to backup %s: %w", backupPath, err)
	}

	return backupPath, nil
}
