package font_test

import (
	"testing"

	"scumm-patcher/internal/font"
)

// minFontData returns a zeroed font buffer large enough to hold all lookup
// addresses needed by SwedishRemapping. The highest address needed is for
// 0xF6 (ö): (0xF6 - 0x20) * 2 + 0x5A = 518, so we need at least 519 bytes.
func minFontData() []byte {
	return make([]byte, 600)
}

// setSwedishSourceGlyphs populates the Windows-1252 source positions used by
// SwedishRemapping so that RemapLookup does not return an error.
func setSwedishSourceGlyphs(data []byte) {
	setGlyph(data, 0xC5, 107) // Å
	setGlyph(data, 0xC4, 106) // Ä
	setGlyph(data, 0xD6, 119) // Ö
	setGlyph(data, 0xE5, 128) // å
	setGlyph(data, 0xE4, 127) // ä
	setGlyph(data, 0xF6, 143) // ö
	setGlyph(data, 0xE9, 132) // é
}

// setGlyph writes a glyph index into the lookup table at the given character code.
func setGlyph(data []byte, code byte, glyphIdx byte) {
	addr := (int(code)-0x20)*2 + 0x5A
	data[addr] = glyphIdx
}

// getGlyph reads the glyph index from the lookup table for the given character code.
func getGlyph(data []byte, code byte) byte {
	addr := (int(code)-0x20)*2 + 0x5A
	return data[addr]
}

// FONT-001: All 7 Swedish characters are remapped from their SCUMM codes to
// their Windows-1252 glyph positions. Å/Ä/Ö/å/ä/ö use SCUMM codes 91–93 and
// 123–125; é uses SCUMM code 130.
func TestRemapLookupSwedish(t *testing.T) {
	data := minFontData()
	setSwedishSourceGlyphs(data)

	out, err := font.RemapLookup(data, font.SwedishRemapping)
	if err != nil {
		t.Fatalf("RemapLookup: %v", err)
	}

	cases := []struct {
		scummCode  byte
		srcCode    byte
		name       string
	}{
		{91, 0xC5, "Å"},
		{92, 0xC4, "Ä"},
		{93, 0xD6, "Ö"},
		{123, 0xE5, "å"},
		{124, 0xE4, "ä"},
		{125, 0xF6, "ö"},
		{130, 0xE9, "é"},
	}
	for _, tc := range cases {
		want := getGlyph(data, tc.srcCode)
		got := getGlyph(out, tc.scummCode)
		if got != want {
			t.Errorf("scumm code %d (%s): glyph = %d, want %d", tc.scummCode, tc.name, got, want)
		}
	}
}

// FONT-002: Input data is not modified (RemapLookup returns a copy).
func TestRemapLookupDoesNotMutateInput(t *testing.T) {
	data := minFontData()
	setGlyph(data, 0xE5, 128)
	original := make([]byte, len(data))
	copy(original, data)

	_, err := font.RemapLookup(data, font.SwedishRemapping)
	if err != nil && err.Error() != "" {
		// Errors from unmapped glyphs are fine here — we only care that data is unchanged.
	}

	for i, b := range data {
		if b != original[i] {
			t.Errorf("input data modified at byte %d: was %d, now %d", i, original[i], b)
		}
	}
}

// FONT-003: Error when a source unicode code has no glyph (index 0).
func TestRemapLookupMissingSourceGlyph(t *testing.T) {
	data := minFontData()
	// Don't set any glyph for 0xE9 (é) — leaves it as 0.

	_, err := font.RemapLookup(data, map[byte]byte{130: 0xE9})
	if err == nil {
		t.Fatal("expected error for unmapped source glyph")
	}
}

// FONT-004: Error when font data is too small for a source lookup address.
func TestRemapLookupDataTooSmallSource(t *testing.T) {
	data := make([]byte, 10) // Way too small.

	_, err := font.RemapLookup(data, map[byte]byte{130: 0xE9})
	if err == nil {
		t.Fatal("expected error for out-of-range source address")
	}
}

// FONT-005: Existing glyph mappings for unrelated characters are preserved.
func TestRemapLookupPreservesOtherEntries(t *testing.T) {
	data := minFontData()
	setGlyph(data, 0xE9, 132) // é — needed for remapping
	setGlyph(data, 'A', 34)   // Regular A — should be untouched

	out, err := font.RemapLookup(data, map[byte]byte{130: 0xE9})
	if err != nil {
		t.Fatalf("RemapLookup: %v", err)
	}

	if got := getGlyph(out, 'A'); got != 34 {
		t.Errorf("glyph for 'A' changed: got %d, want 34", got)
	}
}

// FONT-006: Applying the same remapping twice is idempotent.
func TestRemapLookupIdempotent(t *testing.T) {
	data := minFontData()
	setSwedishSourceGlyphs(data)

	first, err := font.RemapLookup(data, font.SwedishRemapping)
	if err != nil {
		t.Fatalf("first remap: %v", err)
	}
	second, err := font.RemapLookup(first, font.SwedishRemapping)
	if err != nil {
		t.Fatalf("second remap: %v", err)
	}

	for i, b := range first {
		if b != second[i] {
			t.Errorf("not idempotent at byte %d: first=%d, second=%d", i, b, second[i])
		}
	}
}
