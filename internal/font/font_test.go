package font_test

import (
	"testing"

	"scumm-patcher/internal/font"
)

// minFontData returns a zeroed font buffer large enough to hold all lookup
// addresses needed by SwedishRemapping. The highest address needed is for
// 0xF6 (ö): (0xF6 - 0x20) * 2 + 0x5A = 518, so we need at least 519 bytes.
// (The Swedish vowels Å/Ä/Ö/å/ä/ö use their Latin-1 code points as SCUMM codes
// and therefore need no remapping; only é at scumm code 130 → 0xE9 is remapped.)
func minFontData() []byte {
	return make([]byte, 600)
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

// FONT-001: é is remapped from its SCUMM code (130) to its Windows-1252 glyph (0xE9).
// Å/Ä/Ö/å/ä/ö use their Latin-1 code points as SCUMM codes and need no remapping.
func TestRemapLookupSwedish(t *testing.T) {
	data := minFontData()
	setGlyph(data, 0xE9, 132) // é at its Windows-1252 position

	out, err := font.RemapLookup(data, font.SwedishRemapping)
	if err != nil {
		t.Fatalf("RemapLookup: %v", err)
	}

	want := getGlyph(data, 0xE9)
	got := getGlyph(out, 130)
	if got != want {
		t.Errorf("scumm code 130 (é): glyph = %d, want %d", got, want)
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
	setGlyph(data, 0xE9, 132) // é

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
