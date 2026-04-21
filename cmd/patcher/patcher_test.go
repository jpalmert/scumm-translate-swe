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
	w32(1)
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

// fontAddr returns the byte offset within a .font file for the given character code.
func fontAddr(code byte) int { return (int(code)-0x20)*2 + 0x5A }

// writeSETestFixture builds a synthetic PAK with the given magic and entries,
// writes it to a temp dir alongside a translation file, and returns the PAK
// path and translation file path.
func writeSETestFixture(t *testing.T, magic [4]byte, entries []struct{ name, data string }) (pakPath, txPath string) {
	t.Helper()
	raw := buildSyntheticPAK(t, magic, entries)
	dir := t.TempDir()
	pakPath = filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(pakPath, raw, 0644)
	txPath = filepath.Join(dir, "swedish.txt")
	os.WriteFile(txPath, []byte("translation"), 0644)
	return pakPath, txPath
}

var defaultSEEntries = []struct{ name, data string }{
	{"classic/en/monkey1.000", "data000"},
	{"classic/en/monkey1.001", "data001"},
}

// --- SE tests ---

// SE-001: Non-existent input PAK → clear error.
func TestRunSEPatchMissingInput(t *testing.T) {
	dir := t.TempDir()
	txFile := filepath.Join(dir, "swedish.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	if err := runSEPatch("/nonexistent/Monkey1.pak", "", txFile); err == nil {
		t.Fatal("expected error for missing input PAK")
	}
}

// SE-002: Invalid magic in PAK → clear error.
func TestRunSEPatchInvalidMagic(t *testing.T) {
	pakPath, txPath := writeSETestFixture(t, [4]byte{'X', 'P', 'A', 'K'}, defaultSEEntries)

	if err := runSEPatch(pakPath, "", txPath); err == nil {
		t.Fatal("expected error for invalid PAK magic")
	}
}

// SE-003: PAK missing classic/en/monkey1.000 → clear error.
func TestRunSEPatchMissing000(t *testing.T) {
	pakPath, txPath := writeSETestFixture(t, gogMagic, []struct{ name, data string }{
		{"classic/en/monkey1.001", "data001"},
	})

	if err := runSEPatch(pakPath, "", txPath); err == nil {
		t.Fatal("expected error for missing monkey1.000 entry")
	}
}

// SE-004: PAK missing classic/en/monkey1.001 → clear error.
func TestRunSEPatchMissing001(t *testing.T) {
	pakPath, txPath := writeSETestFixture(t, gogMagic, []struct{ name, data string }{
		{"classic/en/monkey1.000", "data000"},
	})

	if err := runSEPatch(pakPath, "", txPath); err == nil {
		t.Fatal("expected error for missing monkey1.001 entry")
	}
}

// SE-005: Translation file not found → clear error.
func TestRunSEPatchMissingTranslation(t *testing.T) {
	pakPath, _ := writeSETestFixture(t, gogMagic, defaultSEEntries)

	if err := runSEPatch(pakPath, "", "/nonexistent/swedish.txt"); err == nil {
		t.Fatal("expected error for missing translation file")
	}
}

// SE-006: In-place mode creates a .bak file before injection.
func TestRunSEPatchInPlaceCreatesBackup(t *testing.T) {
	pakPath, txPath := writeSETestFixture(t, gogMagic, defaultSEEntries)

	runSEPatch(pakPath, "", txPath) //nolint:errcheck — failure expected (fake data)

	if _, err := os.Stat(pakPath + ".bak"); err != nil {
		t.Errorf("backup not created at %s.bak", pakPath)
	}
}

// SE-007: Explicit output path → no backup created for input.
func TestRunSEPatchExplicitOutputNoBackup(t *testing.T) {
	pakPath, txPath := writeSETestFixture(t, gogMagic, defaultSEEntries)
	outPath := filepath.Join(filepath.Dir(pakPath), "Monkey1_patched.pak")

	runSEPatch(pakPath, outPath, txPath) //nolint:errcheck — failure expected (fake data)

	if _, err := os.Stat(pakPath + ".bak"); err == nil {
		t.Error("backup should not be created when explicit output path is given")
	}
}

// SE-010: remapFontEntries patches .font entries and skips others.
func TestRemapFontEntries(t *testing.T) {
	fontData := make([]byte, 600)
	setGlyph := func(data []byte, code byte, idx byte) {
		addr := (int(code)-0x20)*2 + 0x5A
		data[addr] = idx
	}
	// Populate Windows-1252 source positions for all 8 Swedish characters.
	setGlyph(fontData, 0xC5, 107) // Å
	setGlyph(fontData, 0xC4, 106) // Ä
	setGlyph(fontData, 0xD6, 119) // Ö
	setGlyph(fontData, 0xE5, 128) // å
	setGlyph(fontData, 0xE4, 127) // ä
	setGlyph(fontData, 0xF6, 143) // ö
	setGlyph(fontData, 0xE9, 132) // é
	setGlyph(fontData, 0xEA, 133) // ê

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

	// Verify all 8 SCUMM codes are remapped to the correct glyph indices.
	cases := []struct{ scumm, want byte }{
		{91, 107},  // Å
		{92, 106},  // Ä
		{93, 119},  // Ö
		{123, 128}, // å
		{124, 127}, // ä
		{125, 143}, // ö
		{130, 132}, // é
		{136, 133}, // ê
	}
	for _, tc := range cases {
		if got := entries[0].Data[fontAddr(tc.scumm)]; got != tc.want {
			t.Errorf("SCUMM code %d: glyph = %d, want %d", tc.scumm, got, tc.want)
		}
	}
	if !bytes.Equal(entries[1].Data, other) {
		t.Error("non-font entry was modified")
	}
}

// SE-011: remapFontEntries returns error when a font is missing a required glyph.
func TestRemapFontEntriesMissingGlyph(t *testing.T) {
	entries := []*pak.Entry{
		{Name: "fonts/MinisterT_20.font", Data: make([]byte, 600)},
	}
	if _, err := remapFontEntries(entries); err == nil {
		t.Fatal("expected error for font missing required glyphs")
	}
}

// SE-012: remapFontEntries with no .font entries returns 0, nil.
func TestRemapFontEntriesNoFonts(t *testing.T) {
	entries := []*pak.Entry{
		{Name: "classic/en/monkey1.000", Data: []byte("data")},
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

// SE-013: runListPAK succeeds for a valid synthetic PAK.
func TestRunListPAK(t *testing.T) {
	entries := []struct{ name, data string }{
		{"classic/en/monkey1.000", "data000"},
		{"classic/en/monkey1.001", "data001"},
		{"fonts/test.font", "fontdata"},
	}
	raw := buildSyntheticPAK(t, gogMagic, entries)
	dir := t.TempDir()
	pakPath := filepath.Join(dir, "test.pak")
	os.WriteFile(pakPath, raw, 0644)

	if err := runListPAK(pakPath); err != nil {
		t.Fatalf("runListPAK on valid PAK: %v", err)
	}
}

// SE-014: runListPAK returns error for invalid/non-existent file.
func TestRunListPAKInvalidFile(t *testing.T) {
	if err := runListPAK("/nonexistent/file.pak"); err == nil {
		t.Fatal("expected error for non-existent PAK file")
	}
}

// SE-015: runListPAK returns error for file with invalid magic.
func TestRunListPAKBadMagic(t *testing.T) {
	raw := buildSyntheticPAK(t, [4]byte{'B', 'A', 'D', '!'}, defaultSEEntries)
	dir := t.TempDir()
	pakPath := filepath.Join(dir, "bad.pak")
	os.WriteFile(pakPath, raw, 0644)

	if err := runListPAK(pakPath); err == nil {
		t.Fatal("expected error for PAK with invalid magic")
	}
}

// --- SE hint/uitext tests ---

// SE-020: loadUITextTranslation parses tab-separated key-value pairs,
// skipping comments and [E]-prefixed (untranslated) lines.
func TestLoadUITextTranslation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "uitext_swedish.txt")
	content := "# comment\nMENU_BACK\tTillbaka\nMENU_YES\t[E]Yes\n\nMENU_NO\tNej\n"
	os.WriteFile(p, []byte(content), 0644)

	m, err := loadUITextTranslation(p)
	if err != nil {
		t.Fatalf("loadUITextTranslation: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d entries, want 2", len(m))
	}
	if m["MENU_BACK"] != "Tillbaka" {
		t.Errorf("MENU_BACK = %q, want Tillbaka", m["MENU_BACK"])
	}
	if m["MENU_NO"] != "Nej" {
		t.Errorf("MENU_NO = %q, want Nej", m["MENU_NO"])
	}
	if _, ok := m["MENU_YES"]; ok {
		t.Error("[E]-prefixed line should be skipped")
	}
}

// SE-021: loadHintsTranslation parses addr-based hint translations.
func TestLoadHintsTranslation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hints_swedish.txt")
	content := "# comment\n49744\tDu måste använda ölet!\n50192\t[E]Not translated\n50688\tStoppa bröllopet!\n"
	os.WriteFile(p, []byte(content), 0644)

	m, err := loadHintsTranslation(p)
	if err != nil {
		t.Fatalf("loadHintsTranslation: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d entries, want 2", len(m))
	}
	if m[49744] != "Du måste använda ölet!" {
		t.Errorf("addr 49744 = %q", m[49744])
	}
	if m[50688] != "Stoppa bröllopet!" {
		t.Errorf("addr 50688 = %q", m[50688])
	}
	if _, ok := m[50192]; ok {
		t.Error("[E]-prefixed line should be skipped")
	}
}

// SE-022: patchHintEntries with no .hints.csv entry in PAK → skip gracefully.
func TestPatchHintEntriesNoHintsEntry(t *testing.T) {
	dir := t.TempDir()
	hintsPath := filepath.Join(dir, "hints_swedish.txt")
	os.WriteFile(hintsPath, []byte("0\t79\tTest\n"), 0644)

	entries := []*pak.Entry{
		{Name: "classic/en/monkey1.000", Data: []byte("data")},
	}

	n, err := patchHintEntries(entries, hintsPath)
	if err != nil {
		t.Fatalf("patchHintEntries: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 strings replaced, got %d", n)
	}
}

// SE-023: findOptionalSEFile returns empty when file doesn't exist.
func TestFindOptionalSEFileMissing(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "swedish.txt")
	os.WriteFile(base, []byte("x"), 0644)

	if got := findOptionalSEFile(base, "uitext_swedish.txt"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// SE-024: findOptionalSEFile returns path when file exists.
func TestFindOptionalSEFilePresent(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "swedish.txt")
	os.WriteFile(base, []byte("x"), 0644)
	target := filepath.Join(dir, "uitext_swedish.txt")
	os.WriteFile(target, []byte("y"), 0644)

	if got := findOptionalSEFile(base, "uitext_swedish.txt"); got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

// --- Classic tests ---

// CLASSIC-001: Non-existent game directory → clear error.
func TestRunClassicPatchMissingDir(t *testing.T) {
	if err := runClassicPatch("/nonexistent/game/dir", "/dev/null"); err == nil {
		t.Fatal("expected error for missing game dir")
	}
}

// CLASSIC-002: Directory missing MONKEY1.000 → clear error.
func TestRunClassicPatchMissing000(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.001"), []byte("data"), 0644)

	if err := runClassicPatch(dir, "/dev/null"); err == nil {
		t.Fatal("expected error for missing MONKEY1.000")
	}
}

// CLASSIC-003: Directory missing MONKEY1.001 → clear error.
func TestRunClassicPatchMissing001(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), []byte("data"), 0644)

	if err := runClassicPatch(dir, "/dev/null"); err == nil {
		t.Fatal("expected error for missing MONKEY1.001")
	}
}

// CLASSIC-004: Translation file not found → clear error.
func TestRunClassicPatchMissingTranslation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(dir, "MONKEY1.001"), []byte("data"), 0644)

	if err := runClassicPatch(dir, "/nonexistent/swedish.txt"); err == nil {
		t.Fatal("expected error for missing translation file")
	}
}

// CLASSIC-005: Backups are created for both game files with correct content.
func TestRunClassicPatchBackupContent(t *testing.T) {
	dir := t.TempDir()
	orig000 := []byte("original-monkey1-000-data")
	orig001 := []byte("original-monkey1-001-data")
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), orig000, 0644)
	os.WriteFile(filepath.Join(dir, "MONKEY1.001"), orig001, 0644)
	txFile := filepath.Join(dir, "swedish.txt")
	os.WriteFile(txFile, []byte("translation"), 0644)

	runClassicPatch(dir, txFile) //nolint:errcheck — failure expected (fake data)

	bak000, err := os.ReadFile(filepath.Join(dir, "MONKEY1.000.bak"))
	if err != nil {
		t.Fatalf("read .000.bak: %v", err)
	}
	if !bytes.Equal(bak000, orig000) {
		t.Error("MONKEY1.000.bak content differs from original")
	}
	bak001, err := os.ReadFile(filepath.Join(dir, "MONKEY1.001.bak"))
	if err != nil {
		t.Fatalf("read .001.bak: %v", err)
	}
	if !bytes.Equal(bak001, orig001) {
		t.Error("MONKEY1.001.bak content differs from original")
	}
}

// CLASSIC-005c: Lowercase filenames accepted.
func TestFindGameFileLowercase(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "monkey1.000"), []byte("lower"), 0644)

	p, err := findGameFile(dir, "MONKEY1.000", "monkey1.000")
	if err != nil {
		t.Fatalf("findGameFile: %v", err)
	}
	if filepath.Base(p) != "monkey1.000" {
		t.Errorf("expected lowercase path, got %s", p)
	}
}

// CLASSIC-006: Uppercase preferred over lowercase when both exist.
func TestFindGameFileUppercasePreferred(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), []byte("upper"), 0644)
	os.WriteFile(filepath.Join(dir, "monkey1.000"), []byte("lower"), 0644)

	p, err := findGameFile(dir, "MONKEY1.000", "monkey1.000")
	if err != nil {
		t.Fatalf("findGameFile: %v", err)
	}
	if filepath.Base(p) != "MONKEY1.000" {
		t.Errorf("expected uppercase path, got %s", p)
	}
}

// CLASSIC-007: findGameFile returns error when neither name exists.
func TestFindGameFileMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := findGameFile(dir, "MONKEY1.000", "monkey1.000"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// CLASSIC-008: findGameFile accepts MONKEY.000 (alternate naming without "1").
func TestFindGameFileAlternateNaming(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY.000"), []byte("alt"), 0644)

	p, err := findGameFile(dir, "MONKEY1.000", "monkey1.000", "MONKEY.000", "monkey.000")
	if err != nil {
		t.Fatalf("findGameFile: %v", err)
	}
	if filepath.Base(p) != "MONKEY.000" {
		t.Errorf("expected MONKEY.000, got %s", p)
	}
}

// --- Shared ---

// SHARED-001: findTranslationFile returns error for missing explicit path.
func TestFindTranslationFileMissingExplicit(t *testing.T) {
	if _, err := findTranslationFile("/nonexistent/translation.txt"); err == nil {
		t.Fatal("expected error")
	}
}

// SHARED-002: findTranslationFile accepts a valid explicit path.
func TestFindTranslationFileExplicit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "swedish.txt")
	os.WriteFile(p, []byte("translation data"), 0644)

	got, err := findTranslationFile(p)
	if err != nil {
		t.Fatalf("findTranslationFile: %v", err)
	}
	if got != p {
		t.Errorf("got %q, want %q", got, p)
	}
}

// --- Auto-detection tests ---

// DETECT-001: isSEInput returns true for a .pak file.
func TestIsSEInputPAKFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Monkey1.pak")
	os.WriteFile(p, []byte("data"), 0644)
	if !isSEInput(p) {
		t.Error("expected true for .pak file")
	}
}

// DETECT-002: isSEInput returns false for a directory.
func TestIsSEInputDirectory(t *testing.T) {
	dir := t.TempDir()
	if isSEInput(dir) {
		t.Error("expected false for directory")
	}
}

// DETECT-003: isSEInput returns true for a non-existent .pak path (by extension).
func TestIsSEInputNonExistentPAK(t *testing.T) {
	if !isSEInput("/some/path/output.pak") {
		t.Error("expected true for non-existent .pak path")
	}
}

