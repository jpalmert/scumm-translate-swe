//go:build integration

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"scumm-patcher/internal/pak"
)

// repoRoot walks up from the package directory to the repository root
// (identified as the directory containing go.mod).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod in parent directories")
		}
		dir = parent
	}
}

// sePaths returns paths to integration test resources, skipping if missing.
func sePaths(t *testing.T) (pakPath, translationPath string) {
	t.Helper()
	root := repoRoot(t)

	pakPath = filepath.Join(root, "game", "monkey1", "Monkey1.pak")
	translationPath = filepath.Join(root, "translation", "monkey1", "monkey1_swe.txt")

	var missing []string
	for _, p := range []string{pakPath, translationPath} {
		if _, err := os.Stat(p); err != nil {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		t.Skipf("integration files missing:\n  %s", strings.Join(missing, "\n  "))
	}

	// Check scummtr binary is available for the current platform.
	scummtrName := map[string]string{
		"linux":   "scummtr-linux-x64",
		"darwin":  "scummtr-darwin-x64",
		"windows": "scummtr-windows-x64.exe",
	}[runtime.GOOS]
	if scummtrName == "" {
		t.Skipf("integration tests not supported on %s", runtime.GOOS)
	}
	scummtrAsset := filepath.Join(root, "internal/classic/assets", scummtrName)
	if _, err := os.Stat(scummtrAsset); err != nil {
		t.Skipf("scummtr asset missing: %s", scummtrAsset)
	}

	return
}

// INT-SE-001: Full SE pipeline — patched PAK is valid and classic entries grew.
func TestSEPatcherFullPipeline(t *testing.T) {
	pakPath, translationPath := sePaths(t)

	// Read original sizes.
	_, _, _, origEntries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read original: %v", err)
	}
	var orig001Size int
	for _, e := range origEntries {
		if strings.ToLower(e.Name) == "classic/en/monkey1.001" {
			orig001Size = len(e.Data)
			break
		}
	}
	if orig001Size == 0 {
		t.Fatal("classic/en/monkey1.001 not found in original PAK")
	}

	// Run patcher with explicit output (no backup, preserves original).
	dir := t.TempDir()
	outPath := filepath.Join(dir, "Monkey1_patched.pak")

	if err := runSEPatch(pakPath, outPath, translationPath); err != nil {
		t.Fatalf("runSEPatch: %v", err)
	}

	// Verify output is readable.
	_, _, _, patchedEntries, err := pak.Read(outPath)
	if err != nil {
		t.Fatalf("pak.Read patched: %v", err)
	}

	var patched001Size int
	for _, e := range patchedEntries {
		if strings.ToLower(e.Name) == "classic/en/monkey1.001" {
			patched001Size = len(e.Data)
			break
		}
	}
	if patched001Size == 0 {
		t.Fatal("classic/en/monkey1.001 not found in patched PAK")
	}

	if patched001Size <= orig001Size {
		t.Errorf("MONKEY1.001 did not grow: orig=%d, patched=%d", orig001Size, patched001Size)
	}
	t.Logf("MONKEY1.001: %d → %d bytes (+%d)", orig001Size, patched001Size, patched001Size-orig001Size)

	// Entry count must be identical — we only modified data, not structure.
	if len(patchedEntries) != len(origEntries) {
		t.Errorf("entry count changed: orig=%d, patched=%d", len(origEntries), len(patchedEntries))
	}
}

// INT-SE-002: In-place mode creates a backup of Monkey1.pak.
func TestSEPatcherInPlaceBackup(t *testing.T) {
	pakPath, translationPath := sePaths(t)

	// Copy PAK to a temp dir so we can patch it in-place without modifying the original.
	dir := t.TempDir()
	origData, err := os.ReadFile(pakPath)
	if err != nil {
		t.Fatalf("read PAK: %v", err)
	}
	tmpPAK := filepath.Join(dir, "Monkey1.pak")
	if err := os.WriteFile(tmpPAK, origData, 0644); err != nil {
		t.Fatalf("write temp PAK: %v", err)
	}

	// Patch in-place (no explicit output path).
	if err := runSEPatch(tmpPAK, "", translationPath); err != nil {
		t.Fatalf("runSEPatch in-place: %v", err)
	}

	bakPath := tmpPAK + ".bak"
	bakInfo, err := os.Stat(bakPath)
	if err != nil {
		t.Fatalf("backup not created at %s", bakPath)
	}
	if bakInfo.Size() != int64(len(origData)) {
		t.Errorf("backup size mismatch: got %d, want %d", bakInfo.Size(), len(origData))
	}

	bakData, _ := os.ReadFile(bakPath)
	if string(bakData) != string(origData) {
		t.Error("backup content differs from original")
	}
}
