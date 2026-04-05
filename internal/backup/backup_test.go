package backup_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"scumm-patcher/internal/backup"
)

// BACKUP-001: Create makes a byte-identical copy at path+".bak".
func TestCreate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "original.pak")
	content := []byte("original content for testing")
	os.WriteFile(src, content, 0644)

	bakPath, err := backup.Create(src)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	want := src + ".bak"
	if bakPath != want {
		t.Errorf("backup path = %q, want %q", bakPath, want)
	}

	got, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("backup content = %q, want %q", got, content)
	}
}

// BACKUP-002: A second Create call does not overwrite an existing backup.
// This preserves the original file even if Create is called multiple times.
func TestCreateNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "game.pak")
	bak := src + ".bak"

	os.WriteFile(src, []byte("version 1"), 0644)

	// First backup
	if _, err := backup.Create(src); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	// Modify the source to simulate a patched file
	os.WriteFile(src, []byte("version 2 (patched)"), 0644)

	// Second backup call — must NOT overwrite the original backup; returns ErrBackupExists.
	_, err := backup.Create(src)
	if !errors.Is(err, backup.ErrBackupExists) {
		t.Fatalf("second Create: want ErrBackupExists, got %v", err)
	}

	got, _ := os.ReadFile(bak)
	if string(got) != "version 1" {
		t.Errorf("backup was overwritten: got %q, want %q", got, "version 1")
	}
}

// BACKUP-003: Create on a missing source returns an error.
func TestCreateMissingSource(t *testing.T) {
	_, err := backup.Create("/nonexistent/path/to/file.pak")
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}

// BACKUP-004: Backup path is always src+".bak" regardless of content.
func TestCreatePath(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"MONKEY1.000", "MONKEY1.001", "Monkey1.pak"} {
		src := filepath.Join(dir, name)
		os.WriteFile(src, []byte("data"), 0644)

		bakPath, err := backup.Create(src)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if bakPath != src+".bak" {
			t.Errorf("%s: backup path = %q, want %q", name, bakPath, src+".bak")
		}
	}
}
