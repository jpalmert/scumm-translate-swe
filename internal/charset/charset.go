// Package charset patches SCUMM v5 CHAR blocks in MONKEY1.001 to add Swedish
// glyph data. The patched CHAR binaries are embedded at compile time.
//
// MONKEY1.001 is XOR-encoded with key 0x69. This package decodes, patches,
// and re-encodes the file transparently.
//
// Only CHAR_0001 (verb/menu charset) and CHAR_0003 (small text charset) are
// patched; CHAR_0002 (main dialog charset) already contains Swedish glyphs in
// the GOG/Steam SE release.
package charset

import (
	_ "embed"
	"encoding/binary"
	"fmt"
)

const xorKey = 0x69

// Expected sizes for original (unpatched) CHAR blocks. Used to verify that
// the game file matches the version we patched against.
const (
	originalChar0001Size = 2609
	originalChar0003Size = 2071
)

//go:embed assets/char_0001_patched.bin
var patchedChar0001 []byte

//go:embed assets/char_0003_patched.bin
var patchedChar0003 []byte

// PatchMonkey1001 adds Swedish glyph data to CHAR_0001 and CHAR_0003 inside
// the given MONKEY1.001 content (which must be XOR-encoded as stored on disk).
//
// Returns the modified content ready to be written back to disk. If the CHAR
// blocks cannot be found or do not match the expected original sizes, an error
// is returned for CHAR_0001 (critical) or a warning is printed for CHAR_0003
// (non-critical).
func PatchMonkey1001(encoded []byte) ([]byte, error) {
	// Decode in-place copy.
	data := xorDecode(encoded)

	// Apply CHAR_0001 patch (critical — verb/menu text).
	var err error
	data, err = patchCharBlock(data, "CHAR_0001", patchedChar0001, originalChar0001Size)
	if err != nil {
		return nil, err
	}

	// Apply CHAR_0003 patch (best-effort — small text charset).
	data, err = patchCharBlock(data, "CHAR_0003", patchedChar0003, originalChar0003Size)
	if err != nil {
		// Non-critical: log but do not fail. CHAR_0003 is used for minor UI
		// text; the game is playable without it.
		fmt.Printf("    Warning: CHAR_0003 patch skipped: %v\n", err)
	}

	return xorDecode(data), nil // XOR is symmetric; decode = encode
}

// patchCharBlock finds the n-th occurrence of "CHAR" (1-indexed by blockName)
// in data, verifies its size matches originalSize, replaces it with newBlock,
// and updates all ancestor block sizes.
func patchCharBlock(data []byte, blockName string, newBlock []byte, originalSize int) ([]byte, error) {
	// blockName is "CHAR_0001" or "CHAR_0003"; the numeric suffix maps to the
	// 1st and 3rd CHAR block in the file respectively.
	var targetOrdinal int
	switch blockName {
	case "CHAR_0001":
		targetOrdinal = 1
	case "CHAR_0003":
		targetOrdinal = 3
	default:
		return data, fmt.Errorf("unknown block name %q", blockName)
	}

	// Find the target CHAR block by ordinal position.
	offset, err := findCharBlock(data, targetOrdinal)
	if err != nil {
		return data, fmt.Errorf("%s: %w", blockName, err)
	}

	// Verify the original block size.
	foundSize := int(binary.BigEndian.Uint32(data[offset+4:]))
	if foundSize != originalSize {
		return data, fmt.Errorf("%s: expected size %d, got %d (wrong game version?)",
			blockName, originalSize, foundSize)
	}

	// Calculate size delta.
	delta := len(newBlock) - originalSize

	// Splice: replace [offset, offset+originalSize) with newBlock.
	result := make([]byte, len(data)+delta)
	copy(result[:offset], data[:offset])
	copy(result[offset:], newBlock)
	copy(result[offset+len(newBlock):], data[offset+originalSize:])

	// Update ancestor block sizes (LFLF and LECF) by adding delta to their
	// size fields. LECF is always at offset 0; LFLF is the innermost container
	// immediately before the CHAR blocks.
	if delta != 0 {
		if err := updateAncestorSizes(result, offset, delta); err != nil {
			return data, fmt.Errorf("%s: updating sizes: %w", blockName, err)
		}
	}

	return result, nil
}

// findCharBlock returns the byte offset of the n-th "CHAR" block tag (1-based).
func findCharBlock(data []byte, n int) (int, error) {
	count := 0
	for i := 0; i <= len(data)-8; i++ {
		if data[i] == 'C' && data[i+1] == 'H' && data[i+2] == 'A' && data[i+3] == 'R' {
			count++
			if count == n {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("CHAR block %d not found (only found %d)", n, count)
}

// updateAncestorSizes adjusts the size fields of the LECF block (at offset 0)
// and the LFLF block that directly contains the CHAR block at charOffset.
func updateAncestorSizes(data []byte, charOffset, delta int) error {
	// Update LECF at offset 0.
	if len(data) < 8 {
		return fmt.Errorf("file too small for LECF header")
	}
	if string(data[0:4]) != "LECF" {
		return fmt.Errorf("expected LECF at offset 0, got %q", string(data[0:4]))
	}
	addToSize(data, 0, delta)

	// Find the LFLF block whose range contains charOffset.
	lflf, err := findContainingLFLF(data, charOffset)
	if err != nil {
		return err
	}
	addToSize(data, lflf, delta)

	return nil
}

// findContainingLFLF walks the LFLF blocks inside LECF and returns the offset
// of the one whose content includes charOffset.
func findContainingLFLF(data []byte, charOffset int) (int, error) {
	// LECF body starts at offset 8.
	pos := 8
	for pos+8 <= len(data) {
		tag := string(data[pos : pos+4])
		size := int(binary.BigEndian.Uint32(data[pos+4:]))
		if size < 8 {
			break
		}
		if tag == "LFLF" {
			blockEnd := pos + size
			if charOffset >= pos && charOffset < blockEnd {
				return pos, nil
			}
		}
		pos += size
	}
	return 0, fmt.Errorf("no LFLF block contains offset %d", charOffset)
}

// addToSize adds delta to the big-endian uint32 size field at data[offset+4].
func addToSize(data []byte, offset, delta int) {
	current := int(binary.BigEndian.Uint32(data[offset+4:]))
	binary.BigEndian.PutUint32(data[offset+4:], uint32(current+delta))
}

// PatchMonkey1000 updates the charset offset table in MONKEY1.000 to reflect
// the new positions of CHAR blocks after PatchMonkey1001 has been applied.
//
// When CHAR_0001 and CHAR_0003 grow, all charset entries after them in the
// DCHR directory block must have their stored offsets incremented by the
// respective size deltas. The offsets are stored as little-endian uint32
// values in the DCHR block body, relative to the containing LFLF body start.
//
// The function returns the updated encoded content ready to write back to disk.
func PatchMonkey1000(encoded []byte) ([]byte, error) {
	data := xorDecode(encoded)

	// Locate the DCHR directory block.
	dchrOffset := -1
	for i := 0; i <= len(data)-8; i++ {
		if data[i] == 'D' && data[i+1] == 'C' && data[i+2] == 'H' && data[i+3] == 'R' {
			dchrOffset = i
			break
		}
	}
	if dchrOffset < 0 {
		return nil, fmt.Errorf("DCHR block not found in MONKEY1.000")
	}

	blockSize := int(binary.BigEndian.Uint32(data[dchrOffset+4:]))
	body := data[dchrOffset+8 : dchrOffset+blockSize]

	// The DCHR body contains a 2-byte LE entry count followed by count×5-byte
	// entries (1 byte disk-num + 4 bytes LE offset, relative to LFLF body).
	// We update all offsets that fall after char0001RelOffset (the first modified
	// block), incrementing by char0001Delta for offsets after CHAR_0001 and by
	// char0001Delta+char0003Delta for offsets after CHAR_0003.
	//
	// Known original LFLF-body-relative offsets (derived from the GOG/Steam
	// MI1SE release — same classic files in both):
	//   CHAR_0001: 98401   (1st modified block — offset does not change)
	//   CHAR_0002: 101010  → +char0001Delta
	//   CHAR_0003: 105618  → +char0001Delta     (2nd modified block)
	//   CHAR_0004: 107689  → +char0001Delta+char0003Delta
	//   CHAR_0006: 112479  → +char0001Delta+char0003Delta
	const (
		char0001OrigRel = 98401
		char0003OrigRel = 105618
		char0001Delta   = 28 // len(patchedChar0001) - originalChar0001Size
		char0003Delta   = 78 // len(patchedChar0003) - originalChar0003Size
	)

	if len(body) < 2 {
		return nil, fmt.Errorf("DCHR body too short")
	}
	count := int(binary.LittleEndian.Uint16(body[0:2]))
	for i := 0; i < count; i++ {
		entryOffset := 2 + i*5 + 1 // skip 1-byte disk field, read 4-byte LE offset
		if entryOffset+4 > len(body) {
			break
		}
		orig := int(binary.LittleEndian.Uint32(body[entryOffset:]))
		if orig <= char0001OrigRel {
			continue // CHAR_0001 itself or earlier — no shift
		}
		var delta int
		if orig > char0003OrigRel+char0001Delta {
			// Comes after CHAR_0003's (shifted) position
			delta = char0001Delta + char0003Delta
		} else {
			// Between CHAR_0001 and CHAR_0003 (i.e., CHAR_0002 and CHAR_0003 before it grew)
			delta = char0001Delta
		}
		if delta != 0 {
			binary.LittleEndian.PutUint32(body[entryOffset:], uint32(orig+delta))
		}
	}

	return xorDecode(data), nil // XOR is symmetric
}

// xorDecode applies XOR key 0x69 to every byte (symmetric encode/decode).
func xorDecode(in []byte) []byte {
	out := make([]byte, len(in))
	for i, b := range in {
		out[i] = b ^ xorKey
	}
	return out
}
