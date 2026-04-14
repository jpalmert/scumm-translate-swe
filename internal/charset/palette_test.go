package charset

import "testing"

// TestFixCharset6Palette verifies that fixCharset6Palette patches the
// palette bytes at the correct offsets and returns a copy (does not
// mutate the source).
func TestFixCharset6Palette(t *testing.T) {
	// Build a minimal CHAR block: 8-byte header + 14 bytes metadata + 15 bytes palette.
	src := make([]byte, 8+14+15)
	copy(src[0:4], "CHAR")
	// Original English palette for charset 6.
	palette := []byte{9, 10, 11, 12, 13, 14, 15, 2, 14, 0, 1, 0, 0, 0, 0}
	copy(src[charPaletteOffset:], palette)

	fixed := fixCharset6Palette(src)

	// palette[0] → _charsetData[6][1]: must match charsetColor value 6.
	if got := fixed[charPaletteOffset]; got != 6 {
		t.Errorf("palette[0] = %d, want 6", got)
	}
	// palette[1] → _charsetData[6][2]: must match charsetColor value 2.
	if got := fixed[charPaletteOffset+1]; got != 2 {
		t.Errorf("palette[1] = %d, want 2", got)
	}
	// Remaining palette bytes must be unchanged.
	for i := 2; i < 15; i++ {
		if fixed[charPaletteOffset+i] != palette[i] {
			t.Errorf("palette[%d] = %d, want %d", i, fixed[charPaletteOffset+i], palette[i])
		}
	}
	// Source must not be mutated.
	if src[charPaletteOffset] != 9 || src[charPaletteOffset+1] != 10 {
		t.Error("fixCharset6Palette mutated the source slice")
	}
}
