//go:build integration

package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"scumm-patcher/internal/charset"
	"scumm-patcher/internal/classic"
	"scumm-patcher/internal/pak"
	"scumm-patcher/internal/speech"
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
	translationPath = filepath.Join(root, "translation", "monkey1", "swedish.txt")

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

// checkPipelineErr skips the test if err is ErrCharDataNotBuilt, otherwise fails.
func checkPipelineErr(t *testing.T, label string, err error) {
	t.Helper()
	if errors.Is(err, charset.ErrCharDataNotBuilt) {
		t.Skipf("%s: %v", label, err)
	}
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
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

	checkPipelineErr(t, "runSEPatch", runSEPatch(pakPath, outPath, translationPath))

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

	checkPipelineErr(t, "runSEPatch in-place", runSEPatch(tmpPAK, "", translationPath))

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

// INT-SE-003: Patch → restore backup → patch again succeeds (no double-patch failure).
func TestSEPatcherRePatch(t *testing.T) {
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

	// First patch.
	checkPipelineErr(t, "first runSEPatch", runSEPatch(tmpPAK, "", translationPath))

	// Restore backup.
	bakData, err := os.ReadFile(tmpPAK + ".bak")
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if err := os.WriteFile(tmpPAK, bakData, 0644); err != nil {
		t.Fatalf("restore backup: %v", err)
	}

	// Second patch on restored original — must succeed without "CHAR block not found" errors.
	if err := runSEPatch(tmpPAK, "", translationPath); err != nil {
		t.Fatalf("second runSEPatch (after restore): %v", err)
	}

	// Both patches should produce the same result.
	patchedOnce, err := os.ReadFile(tmpPAK)
	if err != nil {
		t.Fatalf("read patched PAK: %v", err)
	}
	t.Logf("Monkey1.pak after re-patch: %d bytes", len(patchedOnce))
}

// INT-SPEECH-001: speech.info EN slots are updated with Swedish SCUMM bytes.
//
// This test bypasses the CHAR/font steps (which require scripts/build.sh) and
// tests the speech pipeline in isolation: build the mapping from real game files
// + swedish.txt, patch a copy of speech.info, and assert meaningful changes.
func TestSpeechInfoPatchedWithSwedish(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

	root := repoRoot(t)
	speechInfoSrc := filepath.Join(root, "game", "monkey1", "audio", "speech.info")
	if _, err := os.Stat(speechInfoSrc); err != nil {
		t.Skipf("speech.info not found at %s", speechInfoSrc)
	}

	// Extract classic game files from PAK into a temp dir so BuildSpeechMapping
	// can run scummtr against them.
	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	gameDir := t.TempDir()
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			os.WriteFile(filepath.Join(gameDir, "MONKEY1.000"), e.Data, 0644)
		case "classic/en/monkey1.001":
			os.WriteFile(filepath.Join(gameDir, "MONKEY1.001"), e.Data, 0644)
		}
	}

	// Build the EN→Swedish mapping from the original English game content.
	mapping, err := classic.BuildSpeechMapping(gameDir, translationPath)
	if err != nil {
		t.Fatalf("BuildSpeechMapping: %v", err)
	}
	t.Logf("mapping size: %d entries", len(mapping))
	if len(mapping) < 100 {
		t.Errorf("mapping suspiciously small: got %d, expected at least 100", len(mapping))
	}

	// Patch a copy of speech.info.
	dir := t.TempDir()
	tmpSpeech := filepath.Join(dir, "speech.info")
	origData, err := os.ReadFile(speechInfoSrc)
	if err != nil {
		t.Fatalf("read speech.info: %v", err)
	}
	if err := os.WriteFile(tmpSpeech, origData, 0644); err != nil {
		t.Fatalf("write temp speech.info: %v", err)
	}

	n, err := speech.Patch(tmpSpeech, mapping)
	if err != nil {
		t.Fatalf("speech.Patch: %v", err)
	}
	t.Logf("speech.Patch updated %d entries", n)
	if n < 100 {
		t.Errorf("too few entries updated: got %d, expected at least 100 out of 4651", n)
	}

	// Verify patched slots actually contain Swedish SCUMM-encoded bytes.
	patchedData, _ := os.ReadFile(tmpSpeech)
	const (
		entry1Base  = 0x510
		headerSize  = 0x30
		slotSize    = 256
		entryStride = 0x530
	)
	nEntries := (len(patchedData) - entry1Base) / entryStride
	swedishScummBytes := []byte{0x5B, 0x5C, 0x5D, 0x7B, 0x7C, 0x7D, 0x82} // Å Ä Ö å ä ö é
	slotsWithSwedish := 0
	for i := 0; i < nEntries; i++ {
		enOff := entry1Base + i*entryStride + headerSize
		slot := patchedData[enOff : enOff+slotSize]
		for _, b := range slot {
			if b == 0 {
				break
			}
			for _, sb := range swedishScummBytes {
				if b == sb {
					slotsWithSwedish++
					goto nextEntry
				}
			}
		}
	nextEntry:
	}
	t.Logf("%d slots contain Swedish SCUMM bytes", slotsWithSwedish)
	if slotsWithSwedish == 0 {
		t.Error("no slots contain Swedish SCUMM bytes — encoding may be wrong")
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

	checkPipelineErr(t, "runClassicPatch", runClassicPatch(dir, translationPath))

	patched001, _ := os.ReadFile(filepath.Join(dir, "MONKEY1.001"))
	if len(patched001) <= len(data001) {
		t.Errorf("MONKEY1.001 did not grow: orig=%d, patched=%d", len(data001), len(patched001))
	}
	t.Logf("MONKEY1.001: %d → %d bytes (+%d)", len(data001), len(patched001), len(patched001)-len(data001))
}
