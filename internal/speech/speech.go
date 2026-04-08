// Package speech patches the speech.info subtitle/audio index used by the
// Secret of Monkey Island Special Edition.
//
// speech.info is a fixed-stride binary file that maps XACT audio cue names
// to subtitle text in five language slots (EN, FR, IT, DE, SP).
//
// The SE engine performs text-based lookup when playing voiced lines in classic
// rendering mode: it reads the current string from the embedded MONKEY1.001,
// searches speech.info's EN slot for a byte-for-byte match, and plays the
// corresponding XACT cue.  After the Swedish translation replaces EN text in
// MONKEY1.001, the EN slots in speech.info must also be updated to match —
// otherwise no audio cue is found and speech is silent.
package speech

import (
	"fmt"
	"os"
)

const (
	entry0Base  = 0x10   // offset of entry 0 (no cue-name header)
	entry1Base  = 0x510  // offset of entry 1 (first cued entry)
	entryStride = 0x530  // bytes per entry for entries 1+
	headerSize  = 0x30   // cue-name header bytes per entry (entries 1+ only)
	slotSize    = 256    // bytes per language slot
)

// Patch updates the English language slot of every entry in the speech.info
// file at path.  For each entry whose EN text exactly matches a key in mapping,
// the slot is replaced with the corresponding value (SCUMM-encoded Swedish bytes).
//
// Returns the number of entries updated and any write error.
func Patch(path string, mapping map[string][]byte) (int, error) {
	if len(mapping) == 0 {
		return 0, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading speech.info: %w", err)
	}

	count := 0

	// Entry 0: no cue-name header, EN slot starts at entry0Base.
	if count0 := patchSlot(data, entry0Base, mapping); count0 > 0 {
		count += count0
	}

	// Entries 1+: each has a headerSize-byte header followed by language slots.
	nEntries := (len(data) - entry1Base) / entryStride
	for i := 0; i < nEntries; i++ {
		enOffset := entry1Base + i*entryStride + headerSize
		count += patchSlot(data, enOffset, mapping)
	}

	if count == 0 {
		return 0, nil
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return 0, fmt.Errorf("writing speech.info: %w", err)
	}
	return count, nil
}

// patchSlot checks the EN slot at enOffset against mapping and replaces it if
// a match is found.  Returns 1 if the slot was updated, 0 otherwise.
func patchSlot(data []byte, enOffset int, mapping map[string][]byte) int {
	if enOffset+slotSize > len(data) {
		return 0
	}
	enText := slotString(data[enOffset : enOffset+slotSize])
	if enText == "" {
		return 0
	}
	sv, ok := mapping[enText]
	if !ok {
		return 0
	}
	writeSlot(data[enOffset:enOffset+slotSize], sv)
	return 1
}

// slotString reads a null-terminated string from a fixed-size slot.
func slotString(slot []byte) string {
	for i, b := range slot {
		if b == 0 {
			return string(slot[:i])
		}
	}
	return string(slot) // no null found — treat entire slot as string
}

// writeSlot writes text into a fixed-size slot, zero-padding the remainder.
func writeSlot(slot []byte, text []byte) {
	for i := range slot {
		slot[i] = 0
	}
	n := len(text)
	if n > len(slot)-1 {
		n = len(slot) - 1 // leave room for null terminator
	}
	copy(slot, text[:n])
}
