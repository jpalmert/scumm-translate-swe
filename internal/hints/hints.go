// Package hints reads, writes, and patches Monkey Island Special Edition
// hint files (hints/monkey1.hints.csv inside the PAK archive).
//
// Despite the .csv extension, the file is a binary format:
//
//	The file has an entry table, an index matrix, and a string pool.
//	The entry table (bytes 0–indexMatrixOffset) encodes a tree of hint groups.
//	The index matrix starts at offset 0x76B0 (MI:SE) and contains 16-byte
//	entries, each with 4 relative uint32 offsets to hint strings (up to 4
//	levels per hint). Entries cycle through 5 languages: EN, FR, DE, IT, ES.
//
//	Strings in the pool are null-terminated Latin-1, padded to 16-byte alignment.
//
// Translation replaces English strings and rebuilds the string pool, updating
// all relative offsets in the index matrix so strings can grow or shrink freely.
package hints

import (
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"
)

const (
	poolAlign = 16

	// indexMatrixOffset is the byte offset where the hint index matrix starts
	// in MI:SE hint files. The first uint32 at this offset, divided by 16,
	// gives the total number of matrix entries (across all 5 languages).
	indexMatrixOffset = 0x76B0

	numLangs          = 5
	maxHintsPerSeries = 4
	matrixEntrySize   = maxHintsPerSeries * 4 // 16 bytes
)

// HintString represents one hint string with its file-absolute address.
type HintString struct {
	Addr uint32 // absolute byte offset in the file
	Text string // decoded UTF-8 text
}

// HintsFile represents a parsed hints binary file.
type HintsFile struct {
	data []byte // full file data
}

// Parse loads a hints binary blob. The data is copied internally.
func Parse(data []byte) (*HintsFile, error) {
	if len(data) < indexMatrixOffset+4 {
		return nil, fmt.Errorf("hints: data too short (%d bytes, need at least %d)", len(data), indexMatrixOffset+4)
	}

	le := binary.LittleEndian
	firstU32 := le.Uint32(data[indexMatrixOffset:])
	numEntries := firstU32 / matrixEntrySize
	if numEntries == 0 {
		return nil, fmt.Errorf("hints: index matrix at 0x%X has 0 entries", indexMatrixOffset)
	}
	if numEntries%numLangs != 0 {
		return nil, fmt.Errorf("hints: entry count %d is not a multiple of %d languages", numEntries, numLangs)
	}

	buf := make([]byte, len(data))
	copy(buf, data)
	return &HintsFile{data: buf}, nil
}

// Serialize returns the (potentially modified) file data.
func (h *HintsFile) Serialize() []byte {
	out := make([]byte, len(h.data))
	copy(out, h.data)
	return out
}

// numMatrixEntries returns the total number of index matrix entries.
func (h *HintsFile) numMatrixEntries() uint32 {
	return binary.LittleEndian.Uint32(h.data[indexMatrixOffset:]) / matrixEntrySize
}

// poolStart returns the absolute offset where the string pool begins
// (right after the index matrix).
func (h *HintsFile) poolStart() uint32 {
	return indexMatrixOffset + h.numMatrixEntries()*matrixEntrySize
}

// matrixFieldAddr returns the absolute address of a specific field in the matrix.
func matrixFieldAddr(entryIdx, hintLevel int) uint32 {
	return indexMatrixOffset + uint32(entryIdx)*matrixEntrySize + uint32(hintLevel)*4
}

// resolveStringAddr reads a relative offset from the index matrix and returns
// the absolute string address. Returns 0 if the relative offset is 0 (empty slot).
func (h *HintsFile) resolveStringAddr(entryIdx, hintLevel int) uint32 {
	fieldAddr := matrixFieldAddr(entryIdx, hintLevel)
	relOff := binary.LittleEndian.Uint32(h.data[fieldAddr:])
	if relOff == 0 {
		return 0
	}
	return relOff + fieldAddr
}

// readStringAt reads a null-terminated Latin-1 string at the given absolute address.
func (h *HintsFile) readStringAt(addr uint32) string {
	end := addr
	for end < uint32(len(h.data)) && h.data[end] != 0 {
		end++
	}
	return decodeLatin1(h.data[addr:end])
}

// ExtractEnglish returns all English hint strings from the file.
// English entries are at matrix indices where index % 5 == 0.
func (h *HintsFile) ExtractEnglish() []HintString {
	numEntries := int(h.numMatrixEntries())
	var result []HintString

	for i := 0; i < numEntries; i += numLangs {
		for level := 0; level < maxHintsPerSeries; level++ {
			addr := h.resolveStringAddr(i, level)
			if addr == 0 || int(addr) >= len(h.data) {
				continue
			}
			text := h.readStringAt(addr)
			if len(text) == 0 {
				continue
			}
			result = append(result, HintString{Addr: addr, Text: text})
		}
	}
	return result
}

// poolString is an internal type for pool rebuild: one string with its
// original absolute address and (possibly replaced) content.
type poolString struct {
	origAddr uint32
	latin1   []byte // encoded content (may be replacement)
}

// ReplaceStrings replaces strings by their absolute address and rebuilds
// the string pool so that strings can grow or shrink freely. All relative
// offsets in the index matrix are updated to point to the new positions.
func (h *HintsFile) ReplaceStrings(replacements map[uint32]string) error {
	numEntries := int(h.numMatrixEntries())
	le := binary.LittleEndian
	ps := h.poolStart()

	// 1. Collect all strings in the pool, ordered by address.
	//    We discover them by walking the index matrix.
	addrSet := make(map[uint32]bool)
	for i := 0; i < numEntries; i++ {
		for level := 0; level < maxHintsPerSeries; level++ {
			addr := h.resolveStringAddr(i, level)
			if addr != 0 && int(addr) < len(h.data) {
				addrSet[addr] = true
			}
		}
	}

	// Sort addresses for sequential pool rebuild.
	addrs := make([]uint32, 0, len(addrSet))
	for a := range addrSet {
		addrs = append(addrs, a)
	}
	sort.Slice(addrs, func(i, j int) bool { return addrs[i] < addrs[j] })

	// 2. Build pool strings, applying replacements.
	strings := make([]poolString, len(addrs))
	for i, addr := range addrs {
		origText := h.readStringAt(addr)
		newText, replaced := replacements[addr]
		if !replaced {
			newText = origText
		}
		latin1, err := utf8ToLatin1(newText)
		if err != nil {
			return fmt.Errorf("hints: addr %d: %w", addr, err)
		}
		strings[i] = poolString{origAddr: addr, latin1: latin1}
	}

	// Check for replacement addresses that don't exist in the pool.
	for addr := range replacements {
		if !addrSet[addr] {
			return fmt.Errorf("hints: replacement address %d not found in string pool", addr)
		}
	}

	// 3. Rebuild the pool with new 16-byte-aligned slots.
	oldToNew := make(map[uint32]uint32, len(strings))
	var newPool []byte
	for i, s := range strings {
		newAddr := ps + uint32(len(newPool))
		oldToNew[s.origAddr] = newAddr

		newPool = append(newPool, s.latin1...)
		newPool = append(newPool, 0) // null terminator
		// Pad to 16-byte alignment (except the very last string — the
		// original file may not pad its final slot).
		if i < len(strings)-1 {
			for len(newPool)%poolAlign != 0 {
				newPool = append(newPool, 0)
			}
		}
	}

	// 4. Build new file: preamble (0..poolStart) + new pool.
	newData := make([]byte, int(ps)+len(newPool))
	copy(newData, h.data[:ps])

	// Write the new pool.
	copy(newData[ps:], newPool)

	// 5. Update all relative offsets in the index matrix.
	for i := 0; i < numEntries; i++ {
		for level := 0; level < maxHintsPerSeries; level++ {
			fieldAddr := matrixFieldAddr(i, level)
			relOff := le.Uint32(newData[fieldAddr:])
			if relOff == 0 {
				continue
			}
			oldAbsAddr := relOff + fieldAddr
			newAbsAddr, ok := oldToNew[oldAbsAddr]
			if !ok {
				return fmt.Errorf("hints: matrix[%d][%d] points to unmapped address %d", i, level, oldAbsAddr)
			}
			newRelOff := newAbsAddr - fieldAddr
			le.PutUint32(newData[fieldAddr:], newRelOff)
		}
	}

	h.data = newData
	return nil
}

func decodeLatin1(b []byte) string {
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}

func utf8ToLatin1(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return nil, fmt.Errorf("invalid UTF-8 at byte %d", i)
		}
		if r > 0xFF {
			return nil, fmt.Errorf("character U+%04X at byte %d is outside Latin-1 range", r, i)
		}
		out = append(out, byte(r))
		i += size
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
