package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"scumm-patcher/internal/pak"
)

// buildSyntheticPAK creates a minimal, valid PAK file for testing.
// Duplicated here from pak_test to avoid a test-only cross-package dependency.
func buildSyntheticPAK(t *testing.T, magic [4]byte, files []struct{ name, data string }) []byte {
	t.Helper()

	const headerSize = 40
	const indexSize = 4
	const entrySize = uint32(20)
	numFiles := uint32(len(files))

	var namesBlob []byte
	namePosMap := make([]uint32, numFiles)
	for i, f := range files {
		namePosMap[i] = uint32(len(namesBlob))
		namesBlob = append(namesBlob, []byte(f.name)...)
		namesBlob = append(namesBlob, 0)
	}

	var dataBlob []byte
	dataPosMap := make([]uint32, numFiles)
	dataSizeMap := make([]uint32, numFiles)
	for i, f := range files {
		dataPosMap[i] = uint32(len(dataBlob))
		dataSizeMap[i] = uint32(len(f.data))
		dataBlob = append(dataBlob, []byte(f.data)...)
	}

	startOfIndex := uint32(headerSize)
	startOfEntries := startOfIndex + indexSize
	startOfNames := startOfEntries + numFiles*entrySize
	startOfData := startOfNames + uint32(len(namesBlob))

	le := binary.LittleEndian
	var buf bytes.Buffer
	w32 := func(v uint32) {
		b := [4]byte{}
		le.PutUint32(b[:], v)
		buf.Write(b[:])
	}

	buf.Write(magic[:])
	w32(1) // version
	w32(startOfIndex)
	w32(startOfEntries)
	w32(startOfNames)
	w32(startOfData)
	w32(indexSize)
	w32(numFiles * entrySize)
	w32(uint32(len(namesBlob)))
	w32(uint32(len(dataBlob)))

	buf.Write(make([]byte, indexSize))

	for i := uint32(0); i < numFiles; i++ {
		w32(dataPosMap[i])
		w32(namePosMap[i])
		w32(dataSizeMap[i])
		w32(dataSizeMap[i])
		w32(0)
	}

	buf.Write(namesBlob)
	buf.Write(dataBlob)
	return buf.Bytes()
}

var gogMagic = [4]byte{'K', 'A', 'P', 'L'}

// SE-001: Non-existent input PAK → clear error.
func TestRunSEPatchMissingInput(t *testing.T) {
	dir := t.TempDir()
	txFile := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	err := runSEPatch("/nonexistent/Monkey1.pak", "", txFile)
	if err == nil {
		t.Fatal("expected error for missing input PAK")
	}
}

// SE-002: Invalid magic in PAK → clear error.
func TestRunSEPatchInvalidMagic(t *testing.T) {
	raw := buildSyntheticPAK(t, [4]byte{'X', 'P', 'A', 'K'}, []struct{ name, data string }{
		{"classic/en/monkey1.000", "data000"},
		{"classic/en/monkey1.001", "data001"},
	})
	dir := t.TempDir()
	inPath := filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(inPath, raw, 0644)

	txFile := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	err := runSEPatch(inPath, "", txFile)
	if err == nil {
		t.Fatal("expected error for invalid PAK magic")
	}
}

// SE-003: PAK missing classic/en/monkey1.000 → clear error.
func TestRunSEPatchMissing000(t *testing.T) {
	raw := buildSyntheticPAK(t, gogMagic, []struct{ name, data string }{
		{"classic/en/monkey1.001", "data001"},
		{"other/asset.dat", "asset"},
	})
	dir := t.TempDir()
	inPath := filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(inPath, raw, 0644)

	txFile := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	err := runSEPatch(inPath, "", txFile)
	if err == nil {
		t.Fatal("expected error for missing monkey1.000 entry")
	}
}

// SE-004: PAK missing classic/en/monkey1.001 → clear error.
func TestRunSEPatchMissing001(t *testing.T) {
	raw := buildSyntheticPAK(t, gogMagic, []struct{ name, data string }{
		{"classic/en/monkey1.000", "data000"},
		{"other/asset.dat", "asset"},
	})
	dir := t.TempDir()
	inPath := filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(inPath, raw, 0644)

	txFile := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	err := runSEPatch(inPath, "", txFile)
	if err == nil {
		t.Fatal("expected error for missing monkey1.001 entry")
	}
}

// SE-005: Translation file not found → clear error.
func TestRunSEPatchMissingTranslation(t *testing.T) {
	raw := buildSyntheticPAK(t, gogMagic, []struct{ name, data string }{
		{"classic/en/monkey1.000", "data000"},
		{"classic/en/monkey1.001", "data001"},
	})
	dir := t.TempDir()
	inPath := filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(inPath, raw, 0644)

	err := runSEPatch(inPath, "", "/nonexistent/monkey1_swe.txt")
	if err == nil {
		t.Fatal("expected error for missing translation file")
	}
}

// SE-006: In-place mode creates a .bak file.
// The injection itself will fail (fake game data), but backup must be created
// before injection is attempted.
func TestRunSEPatchInPlaceCreatesBackup(t *testing.T) {
	raw := buildSyntheticPAK(t, gogMagic, []struct{ name, data string }{
		{"classic/en/monkey1.000", "data000"},
		{"classic/en/monkey1.001", "data001"},
	})
	dir := t.TempDir()
	inPath := filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(inPath, raw, 0644)

	txFile := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	// We expect runSEPatch to fail (scummtr can't handle fake data), but the
	// backup should be created before the injection step.
	runSEPatch(inPath, "", txFile) //nolint:errcheck — failure expected

	bakPath := inPath + ".bak"
	if _, err := os.Stat(bakPath); err != nil {
		t.Errorf("backup not created at %s", bakPath)
	}
}

// SE-007: Explicit output path → no backup created for input.
func TestRunSEPatchExplicitOutputNoBackup(t *testing.T) {
	raw := buildSyntheticPAK(t, gogMagic, []struct{ name, data string }{
		{"classic/en/monkey1.000", "data000"},
		{"classic/en/monkey1.001", "data001"},
	})
	dir := t.TempDir()
	inPath := filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(inPath, raw, 0644)

	txFile := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	outPath := filepath.Join(dir, "Monkey1_patched.pak")

	runSEPatch(inPath, outPath, txFile) //nolint:errcheck — failure expected

	bakPath := inPath + ".bak"
	if _, err := os.Stat(bakPath); err == nil {
		t.Error("backup should not be created when an explicit output path is given")
	}
}

// SE-010: remapFontEntries patches .font entries and skips non-font entries.
func TestRemapFontEntries(t *testing.T) {
	// Build a minimal font buffer with Swedish glyphs at Windows-1252 positions.
	fontData := make([]byte, 600)
	setGlyph := func(data []byte, code byte, idx byte) {
		addr := (int(code)-0x20)*2 + 0x5A
		data[addr] = idx
	}
	setGlyph(fontData, 0xC5, 107) // Å
	setGlyph(fontData, 0xC4, 106) // Ä
	setGlyph(fontData, 0xD6, 119) // Ö
	setGlyph(fontData, 0xE5, 128) // å
	setGlyph(fontData, 0xE4, 127) // ä
	setGlyph(fontData, 0xF6, 143) // ö
	setGlyph(fontData, 0xE9, 132) // é

	other := []byte("not a font file")

	entries := []*pak.Entry{
		{Name: "fonts/MinisterT_20.font", Data: append([]byte(nil), fontData...)},
		{Name: "other/asset.dat", Data: append([]byte(nil), other...)},
	}

	count, err := remapFontEntries(entries)
	if err != nil {
		t.Fatalf("remapFontEntries: %v", err)
	}
	if count != 1 {
		t.Errorf("patched %d font files, want 1", count)
	}

	// Å (SCUMM code 91) should now point to glyph 107.
	fontAddr := func(code byte) int { return (int(code)-0x20)*2 + 0x5A }
	if got := entries[0].Data[fontAddr(91)]; got != 107 {
		t.Errorf("SCUMM code 91 (Å): glyph = %d, want 107", got)
	}
	// å (SCUMM code 123) should now point to glyph 128.
	if got := entries[0].Data[fontAddr(123)]; got != 128 {
		t.Errorf("SCUMM code 123 (å): glyph = %d, want 128", got)
	}
	// Non-font entry must be unchanged.
	if !bytes.Equal(entries[1].Data, other) {
		t.Error("non-font entry was modified")
	}
}

// SE-011: remapFontEntries returns error when a font is missing a required glyph.
func TestRemapFontEntriesMissingGlyph(t *testing.T) {
	// Font data with no glyphs set — all indices are 0.
	entries := []*pak.Entry{
		{Name: "fonts/MinisterT_20.font", Data: make([]byte, 600)},
	}
	_, err := remapFontEntries(entries)
	if err == nil {
		t.Fatal("expected error for font missing required glyphs")
	}
}

// SE-012: remapFontEntries with no .font entries returns 0, nil (not an error).
// This covers the case where a PAK has no font files — graceful no-op.
func TestRemapFontEntriesNoFonts(t *testing.T) {
	entries := []*pak.Entry{
		{Name: "classic/en/monkey1.000", Data: []byte("data")},
		{Name: "classic/en/monkey1.001", Data: []byte("data")},
		{Name: "other/asset.dat", Data: []byte("asset")},
	}
	count, err := remapFontEntries(entries)
	if err != nil {
		t.Fatalf("remapFontEntries with no fonts: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 fonts patched, got %d", count)
	}
}

// SE-008: findTranslationFile returns error for missing explicit path.
func TestFindTranslationFileMissingExplicit(t *testing.T) {
	_, err := findTranslationFile("/nonexistent/translation.txt")
	if err == nil {
		t.Fatal("expected error")
	}
}

// SE-009: findTranslationFile accepts a valid explicit path.
func TestFindTranslationFileExplicit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(p, []byte("translation data"), 0644)

	got, err := findTranslationFile(p)
	if err != nil {
		t.Fatalf("findTranslationFile: %v", err)
	}
	if got != p {
		t.Errorf("got %q, want %q", got, p)
	}
}
