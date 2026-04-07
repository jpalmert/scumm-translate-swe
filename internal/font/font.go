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

// SwedishRemapping maps SCUMM internal codes to the Windows-1252 positions
// that already carry the correct glyphs in the SE .font files.
//
// The classic SCUMM engine uses custom codes 91–93 and 123–125 for the Swedish
// capital and lowercase vowels (replacing ASCII punctuation that never appears
// in dialog). The SE .font files follow Windows-1252, so those glyphs live at
// their native Latin-1 positions. RemapLookup copies each glyph to the SCUMM
// code so the SE new-graphics renderer finds them at the expected positions.
var SwedishRemapping = map[byte]byte{
	91:  0xC5, // Å
	92:  0xC4, // Ä
	93:  0xD6, // Ö
	123: 0xE5, // å
	124: 0xE4, // ä
	125: 0xF6, // ö
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
