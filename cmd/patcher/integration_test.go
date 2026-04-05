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

// repoRoot walks up from the package directory to the repository root.
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

// integrationPaths returns test resource paths, skipping if any are missing.
func integrationPaths(t *testing.T) (pakPath, translationPath string) {
	t.Helper()
	root := repoRoot(t)

	pakPath = filepath.Join(root, "game", "monkey1", "Monkey1.pak")
	translationPath = filepath.Join(root, "translation", "monkey1", "monkey1.txt")

	var missing []string
	for _, p := range []string{pakPath, translationPath} {
		if _, err := os.Stat(p); err != nil {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		t.Skipf("integration files missing:\n  %s", strings.Join(missing, "\n  "))
	}

	scummtrName := map[string]string{
		"linux":   "scummtr-linux-x64",
		"darwin":  "scummtr-darwin-x64",
		"windows": "scummtr-windows-x64.exe",
	}[runtime.GOOS]
	if scummtrName == "" {
		t.Skipf("integration tests not supported on %s", runtime.GOOS)
	}
	if _, err := os.Stat(filepath.Join(root, "internal/classic/assets", scummtrName)); err != nil {
		t.Skipf("scummtr asset missing: %s", scummtrName)
	}
	return
}

// INT-SE-001: Full SE pipeline — patched PAK is valid, .001 grew, fonts patched.
func TestSEPatcherFullPipeline(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

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

	dir := t.TempDir()
	outPath := filepath.Join(dir, "Monkey1_patched.pak")

	if err := runSEPatch(pakPath, outPath, translationPath); err != nil {
		t.Fatalf("runSEPatch: %v", err)
	}

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

	if len(patchedEntries) != len(origEntries) {
		t.Errorf("entry count changed: orig=%d, patched=%d", len(origEntries), len(patchedEntries))
	}

	fontAddr := func(code byte) int { return (int(code)-0x20)*2 + 0x5A }
	fontPatched := 0
	for _, e := range patchedEntries {
		if !strings.HasSuffix(strings.ToLower(e.Name), ".font") {
			continue
		}
		if len(e.Data) > fontAddr(91) && e.Data[fontAddr(91)] != 0 {
			fontPatched++
		}
	}
	if fontPatched == 0 {
		t.Error("no .font entries have SCUMM code 91 (Å) remapped — font patching may not have run")
	}
	t.Logf("%d .font entries have Å remapped", fontPatched)
}

// INT-SE-002: In-place mode creates backup with correct content.
func TestSEPatcherInPlaceBackup(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

	dir := t.TempDir()
	origData, err := os.ReadFile(pakPath)
	if err != nil {
		t.Fatalf("read PAK: %v", err)
	}
	tmpPAK := filepath.Join(dir, "Monkey1.pak")
	if err := os.WriteFile(tmpPAK, origData, 0644); err != nil {
		t.Fatalf("write temp PAK: %v", err)
	}

	if err := runSEPatch(tmpPAK, "", translationPath); err != nil {
		t.Fatalf("runSEPatch in-place: %v", err)
	}

	bakData, err := os.ReadFile(tmpPAK + ".bak")
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if len(bakData) != len(origData) {
		t.Errorf("backup size mismatch: got %d, want %d", len(bakData), len(origData))
	}
	if string(bakData) != string(origData) {
		t.Error("backup content differs from original")
	}
}

// INT-CLASSIC: Real Swedish translation grows .001.
func TestClassicPatcherFullPipeline(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	var data000, data001 []byte
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			data000 = append([]byte(nil), e.Data...)
		case "classic/en/monkey1.001":
			data001 = append([]byte(nil), e.Data...)
		}
	}
	if data000 == nil || data001 == nil {
		t.Fatal("classic files not found in PAK")
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), data000, 0644)
	os.WriteFile(filepath.Join(dir, "MONKEY1.001"), data001, 0644)

	if err := runClassicPatch(dir, translationPath); err != nil {
		t.Fatalf("runClassicPatch: %v", err)
	}

	patched001, _ := os.ReadFile(filepath.Join(dir, "MONKEY1.001"))
	if len(patched001) <= len(data001) {
		t.Errorf("MONKEY1.001 did not grow: orig=%d, patched=%d", len(data001), len(patched001))
	}
	t.Logf("MONKEY1.001: %d → %d bytes (+%d)", len(data001), len(patched001), len(patched001)-len(data001))
}
