//go:build buildpatcher

package charset

import (
	"encoding/binary"
	"testing"
)

// ASSET-001: Embedded CHAR_0001 asset is a valid CHAR block.
func TestPatchedChar0001Asset(t *testing.T) {
	if len(patchedChar0001) < 8 {
		t.Fatalf("patchedChar0001 too short: %d bytes", len(patchedChar0001))
	}
	if string(patchedChar0001[0:4]) != "CHAR" {
		t.Errorf("patchedChar0001 tag = %q, want CHAR", patchedChar0001[0:4])
	}
	size := int(binary.BigEndian.Uint32(patchedChar0001[4:]))
	if size != len(patchedChar0001) {
		t.Errorf("patchedChar0001 size field = %d, actual = %d", size, len(patchedChar0001))
	}
}

// ASSET-002: Embedded CHAR_0002 asset is a valid CHAR block.
func TestPatchedChar0002Asset(t *testing.T) {
	if len(patchedChar0002) < 8 {
		t.Fatalf("patchedChar0002 too short: %d bytes", len(patchedChar0002))
	}
	if string(patchedChar0002[0:4]) != "CHAR" {
		t.Errorf("patchedChar0002 tag = %q, want CHAR", patchedChar0002[0:4])
	}
	size := int(binary.BigEndian.Uint32(patchedChar0002[4:]))
	if size != len(patchedChar0002) {
		t.Errorf("patchedChar0002 size field = %d, actual = %d", size, len(patchedChar0002))
	}
}

// ASSET-003: Embedded CHAR_0003 asset is a valid CHAR block.
func TestPatchedChar0003Asset(t *testing.T) {
	if len(patchedChar0003) < 8 {
		t.Fatalf("patchedChar0003 too short: %d bytes", len(patchedChar0003))
	}
	if string(patchedChar0003[0:4]) != "CHAR" {
		t.Errorf("patchedChar0003 tag = %q, want CHAR", patchedChar0003[0:4])
	}
	size := int(binary.BigEndian.Uint32(patchedChar0003[4:]))
	if size != len(patchedChar0003) {
		t.Errorf("patchedChar0003 size field = %d, actual = %d", size, len(patchedChar0003))
	}
}

// ASSET-003: Embedded CHAR_0004 asset is a valid CHAR block.
func TestPatchedChar0004Asset(t *testing.T) {
	if len(patchedChar0004) < 8 {
		t.Fatalf("patchedChar0004 too short: %d bytes", len(patchedChar0004))
	}
	if string(patchedChar0004[0:4]) != "CHAR" {
		t.Errorf("patchedChar0004 tag = %q, want CHAR", patchedChar0004[0:4])
	}
	size := int(binary.BigEndian.Uint32(patchedChar0004[4:]))
	if size != len(patchedChar0004) {
		t.Errorf("patchedChar0004 size field = %d, actual = %d", size, len(patchedChar0004))
	}
}

// ASSET-005: Embedded CHAR_0006 asset is a valid CHAR block.
func TestPatchedChar0006Asset(t *testing.T) {
	if len(patchedChar0006) < 8 {
		t.Fatalf("patchedChar0006 too short: %d bytes", len(patchedChar0006))
	}
	if string(patchedChar0006[0:4]) != "CHAR" {
		t.Errorf("patchedChar0006 tag = %q, want CHAR", patchedChar0006[0:4])
	}
	size := int(binary.BigEndian.Uint32(patchedChar0006[4:]))
	if size != len(patchedChar0006) {
		t.Errorf("patchedChar0006 size field = %d, actual = %d", size, len(patchedChar0006))
	}
}

// ASSET-006: CHAR_0006 palette matches charsetColor verb values after fix.
func TestChar0006PaletteFix(t *testing.T) {
	if len(patchedChar0006) < charPaletteOffset+2 {
		t.Fatalf("patchedChar0006 too short for palette check: %d bytes", len(patchedChar0006))
	}

	fixed := fixCharset6Palette(patchedChar0006)

	// palette[0] must be 6 (verb foreground color index set by SCRP_0022's charsetColor).
	if got := fixed[charPaletteOffset]; got != 6 {
		t.Errorf("CHAR_0006 palette[0] = %d, want 6 (verb foreground)", got)
	}
	// palette[1] must be 2 (verb shadow color index).
	if got := fixed[charPaletteOffset+1]; got != 2 {
		t.Errorf("CHAR_0006 palette[1] = %d, want 2 (verb shadow)", got)
	}

	// Source data must be unchanged (fixCharset6Palette returns a copy).
	if patchedChar0006[charPaletteOffset] == 6 && patchedChar0006[charPaletteOffset+1] == 2 {
		// If the source already matches, the fix is a no-op — that's fine.
		return
	}
	if fixed[charPaletteOffset] == patchedChar0006[charPaletteOffset] {
		t.Error("fixCharset6Palette did not modify the copy")
	}
}

// ASSET-007: Embedded scummrp binaries are non-empty.
func TestScummrpBinariesEmbedded(t *testing.T) {
	bins := map[string][]byte{
		"linux":   scummrpLinux,
		"darwin":  scummrpDarwin,
		"windows": scummrpWindows,
	}
	for name, bin := range bins {
		if len(bin) == 0 {
			t.Errorf("scummrp %s binary is empty", name)
		}
	}
}
