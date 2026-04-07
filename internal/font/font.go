// Package font patches the ASCII→glyph lookup table in MI1SE .font files.
//
// The SE engine stores dialog text using byte values written by scummtr
// (SCUMM internal codes), but the SE .font files use Windows-1252 codes
// for their glyph mappings. Swedish characters are therefore rendered as
// punctuation unless the lookup table is updated.
//
// RemapLookup fixes this by pointing each SCUMM internal code to the same
// glyph already used by the corresponding Windows-1252 code. No new glyph
// images are added — this is a pure lookup-table patch.
//
// .font lookup table layout (from font.py / MISETranslator):
//
//	address of code c: (c - 0x20) * 2 + 0x5A
//	each entry is 2 bytes; only the first byte is the glyph index.
package font

import "fmt"

const (
	lookupBase = 0x5A
	lookupStep = 2
)

// SwedishRemapping maps SCUMM internal codes (as stored by scummtr -c) to the
// Windows-1252 codes that already have correct glyphs in the SE font files.
// Swedish characters use their native Latin-1/Windows-1252 code points
// (Å=197, Ä=196, Ö=214, å=229, ä=228, ö=246), so the SE font already has
// correct glyphs at those positions — no remapping needed for them.
// Only é (SCUMM code 130) needs remapping to its Windows-1252 position 0xE9.
var SwedishRemapping = map[byte]byte{
	130: 0xE9, // é
}

// RemapLookup updates the glyph lookup table in a .font file so that each
// scummCode entry points to the same glyph as its paired unicodeCode.
//
// Returns a modified copy of data; the input slice is not modified.
// Returns an error if any source glyph index is 0 (unmapped) or if any
// lookup address falls outside the font data.
func RemapLookup(data []byte, remappings map[byte]byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)

	for scummCode, unicodeCode := range remappings {
		srcAddr := lookupAddr(unicodeCode)
		dstAddr := lookupAddr(scummCode)

		if srcAddr+1 > len(out) {
			return nil, fmt.Errorf(
				"unicode code 0x%02X: lookup address %d out of range (font size %d)",
				unicodeCode, srcAddr, len(out))
		}
		if dstAddr+1 > len(out) {
			return nil, fmt.Errorf(
				"scumm code %d: lookup address %d out of range (font size %d)",
				scummCode, dstAddr, len(out))
		}

		glyphIdx := out[srcAddr]
		if glyphIdx == 0 {
			return nil, fmt.Errorf(
				"unicode code 0x%02X has no glyph in this font (index 0)",
				unicodeCode)
		}

		out[dstAddr] = glyphIdx
	}

	return out, nil
}

func lookupAddr(code byte) int {
	return (int(code)-0x20)*lookupStep + lookupBase
}
