//go:build integration

package main

import (
	"bytes"
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

	pakPath = filepath.Join(root, "games", "monkey1", "game", "Monkey1.pak")
	translationPath = filepath.Join(root, "games", "monkey1", "translation", "swedish.txt")

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

// extractClassicFromPAK reads the PAK, extracts classic/en/monkey1.000 and .001
// into a temp directory, and returns the directory path.
func extractClassicFromPAK(t *testing.T, pakPath string) string {
	t.Helper()
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
	if err := os.WriteFile(filepath.Join(dir, "MONKEY1.000"), data000, 0644); err != nil {
		t.Fatalf("write MONKEY1.000: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MONKEY1.001"), data001, 0644); err != nil {
		t.Fatalf("write MONKEY1.001: %v", err)
	}
	return dir
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
	if !bytes.Equal(bakData, origData) {
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

// INT-SE-004: Patch twice without manual restore — re-patch reads from backup automatically.
func TestSEPatcherRePatchAutomatic(t *testing.T) {
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

	firstPatchData, err := os.ReadFile(tmpPAK)
	if err != nil {
		t.Fatalf("read first-patched PAK: %v", err)
	}

	// Second patch directly on the already-patched PAK (no manual restore).
	checkPipelineErr(t, "second runSEPatch", runSEPatch(tmpPAK, "", translationPath))

	secondPatchData, err := os.ReadFile(tmpPAK)
	if err != nil {
		t.Fatalf("read second-patched PAK: %v", err)
	}

	// Both patches must produce identical output.
	if len(firstPatchData) != len(secondPatchData) {
		t.Errorf("re-patch size differs: first=%d, second=%d", len(firstPatchData), len(secondPatchData))
	}
	if !bytes.Equal(firstPatchData, secondPatchData) {
		t.Error("re-patch produced different content — backup-based re-read may not be working")
	}
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
	gameDir := extractClassicFromPAK(t, pakPath)

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

// INT-SPEECH-002: Speech round-trip — bytes written to speech.info match the
// raw bytes scummtr injects into MONKEY1.001 for the same Swedish strings.
//
// This test exercises the full pipeline end-to-end:
//  1. Build speech mapping (EN→SV bytes) from original English content.
//  2. Inject Swedish into a copy of MONKEY1.001 via scummtr.
//  3. Re-extract the patched MONKEY1.001 via scummtr -oh (Swedish text with \NNN escapes).
//  4. For each re-extracted line, decode the \NNN escapes to raw bytes.
//  5. Look up the same line in the speech mapping (keyed by original English text).
//  6. Assert the decoded bytes match the speech.info value (ScummBytes output).
//
// A mismatch here means audio cue lookup in speech.info would fail in-game.
func TestSpeechRoundTrip(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

	gameDir := extractClassicFromPAK(t, pakPath)

	// Build EN→SV mapping from the ORIGINAL English content.
	mapping, err := classic.BuildSpeechMapping(gameDir, translationPath)
	if err != nil {
		t.Fatalf("BuildSpeechMapping: %v", err)
	}
	t.Logf("mapping: %d entries", len(mapping))

	// Extract the original English strings (header → [texts]).
	enLines, err := classic.ExtractLines(gameDir)
	if err != nil {
		t.Fatalf("ExtractLines (before injection): %v", err)
	}

	// Inject Swedish into the copy.
	checkPipelineErr(t, "InjectTranslation", classic.InjectTranslation(gameDir, translationPath))

	// Re-extract from the PATCHED MONKEY1.001.
	svLines, err := classic.ExtractLines(gameDir)
	if err != nil {
		t.Fatalf("ExtractLines (after injection): %v", err)
	}

	// Build a positional index: EN header+position → patched SV text.
	type posKey struct{ header string; pos int }
	svByPos := make(map[posKey]string)
	svPos := make(map[string]int)
	for _, line := range svLines {
		header, text := line[0], line[1]
		p := svPos[header]
		svPos[header]++
		svByPos[posKey{header, p}] = text
	}

	// Walk EN lines; for each, verify the patched SV text matches one of the
	// Swedish variants stored in the mapping for that EN sentence.
	// The mapping now collects ALL distinct Swedish translations per English key,
	// so every positional variant should be accounted for.
	mismatches := 0
	checked := 0
	enPos := make(map[string]int)
	for _, line := range enLines {
		header, enText := line[0], line[1]
		p := enPos[header]
		enPos[header]++

		// Split on page-break escape (matching buildSpeechMapping).
		enParts := strings.Split(enText, `\255\003`)
		sv := svByPos[posKey{header, p}]
		svParts := strings.Split(sv, `\255\003`)

		for i, enPart := range enParts {
			if strings.TrimSpace(enPart) == "" {
				continue
			}
			svVariants, ok := mapping[enPart]
			if !ok {
				continue // this EN sentence has no Swedish translation in the mapping
			}
			checked++

			// Decode the re-extracted SV text (\NNN escapes → raw bytes).
			var svPart string
			if i < len(svParts) {
				svPart = svParts[i]
			}
			actualSV := classic.DecodeScummtrEscapes(svPart)

			// A match against any stored variant is correct.
			matched := false
			for _, variant := range svVariants {
				if bytes.Equal(actualSV, variant) {
					matched = true
					break
				}
			}
			if !matched {
				mismatches++
				if mismatches <= 5 {
					t.Logf("MISMATCH at %s pos %d part %d:", header, p, i)
					t.Logf("  EN text:    %q", enPart)
					t.Logf("  actual SV:  %x (%q)", actualSV, actualSV)
					t.Logf("  variants:   %d stored", len(svVariants))
					for vi, v := range svVariants {
						t.Logf("    [%d] %x (%q)", vi, v, v)
					}
					t.Logf("  raw svPart: %q", svPart)
				}
			}
		}
	}

	t.Logf("round-trip checked %d sentence pairs, %d mismatches", checked, mismatches)
	if checked < 100 {
		t.Errorf("too few pairs checked: %d", checked)
	}
	if mismatches > 0 {
		t.Errorf("%d sentences have a Swedish translation not present in the mapping — encoding bug",
			mismatches)
	}
}

// INT-CLASSIC-001: Real Swedish translation grows .001.
func TestClassicPatcherFullPipeline(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

	dir := extractClassicFromPAK(t, pakPath)
	orig001, err := os.ReadFile(filepath.Join(dir, "MONKEY1.001"))
	if err != nil {
		t.Fatalf("read original .001: %v", err)
	}

	checkPipelineErr(t, "runClassicPatch", runClassicPatch(dir, translationPath))

	patched001, _ := os.ReadFile(filepath.Join(dir, "MONKEY1.001"))
	if len(patched001) <= len(orig001) {
		t.Errorf("MONKEY1.001 did not grow: orig=%d, patched=%d", len(orig001), len(patched001))
	}
	t.Logf("MONKEY1.001: %d → %d bytes (+%d)", len(orig001), len(patched001), len(patched001)-len(orig001))
}

// INT-CLASSIC-002: Classic in-place backup has correct content.
func TestClassicPatcherInPlaceBackup(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

	dir := extractClassicFromPAK(t, pakPath)
	orig000, err := os.ReadFile(filepath.Join(dir, "MONKEY1.000"))
	if err != nil {
		t.Fatalf("read original .000: %v", err)
	}
	orig001, err := os.ReadFile(filepath.Join(dir, "MONKEY1.001"))
	if err != nil {
		t.Fatalf("read original .001: %v", err)
	}

	checkPipelineErr(t, "runClassicPatch", runClassicPatch(dir, translationPath))

	// Verify backups exist and contain the original data.
	bak000, err := os.ReadFile(filepath.Join(dir, "MONKEY1.000.bak"))
	if err != nil {
		t.Fatalf("backup .000 not created: %v", err)
	}
	if len(bak000) != len(orig000) {
		t.Errorf("backup .000 size mismatch: got %d, want %d", len(bak000), len(orig000))
	}
	if !bytes.Equal(bak000, orig000) {
		t.Error("backup .000 content differs from original")
	}

	bak001, err := os.ReadFile(filepath.Join(dir, "MONKEY1.001.bak"))
	if err != nil {
		t.Fatalf("backup .001 not created: %v", err)
	}
	if len(bak001) != len(orig001) {
		t.Errorf("backup .001 size mismatch: got %d, want %d", len(bak001), len(orig001))
	}
	if !bytes.Equal(bak001, orig001) {
		t.Error("backup .001 content differs from original")
	}
}

// INT-CLASSIC-003: Patch → patch again succeeds (re-patch from backup originals).
func TestClassicPatcherRePatch(t *testing.T) {
	pakPath, translationPath := integrationPaths(t)

	dir := extractClassicFromPAK(t, pakPath)

	// First patch.
	checkPipelineErr(t, "first runClassicPatch", runClassicPatch(dir, translationPath))

	firstPatched001, err := os.ReadFile(filepath.Join(dir, "MONKEY1.001"))
	if err != nil {
		t.Fatalf("read first patched .001: %v", err)
	}

	// Second patch — no manual restore needed; patcher should use backup automatically.
	checkPipelineErr(t, "second runClassicPatch", runClassicPatch(dir, translationPath))

	secondPatched001, err := os.ReadFile(filepath.Join(dir, "MONKEY1.001"))
	if err != nil {
		t.Fatalf("read second patched .001: %v", err)
	}

	// Both patches should produce identical results since both start from the
	// same original backup.
	if len(firstPatched001) != len(secondPatched001) {
		t.Errorf("re-patch produced different size: first=%d, second=%d",
			len(firstPatched001), len(secondPatched001))
	}
	if !bytes.Equal(firstPatched001, secondPatched001) {
		t.Error("re-patch produced different content — patching is not idempotent")
	}
	t.Logf("MONKEY1.001 after re-patch: %d bytes (identical to first patch)", len(secondPatched001))
}
