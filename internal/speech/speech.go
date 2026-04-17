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
//
// When the same English phrase has multiple distinct Swedish translations (e.g.
// the same line used in different contexts), Patch appends duplicate entries to
// speech.info — each with the same XACT cue header but a different EN slot —
// so the engine can match any of the Swedish variants to the audio cue.
package speech

import (
	"fmt"
	"os"
)

const (
	entry0Base  = 0x10  // offset of entry 0 (no cue-name header)
	entry1Base  = 0x510 // offset of entry 1 (first cued entry)
	entryStride = 0x530 // bytes per entry for entries 1+
	headerSize  = 0x30  // cue-name header bytes per entry (entries 1+ only)
	slotSize    = 256   // bytes per language slot
)

// Patch updates the English language slot of every entry in the speech.info
// file at path. For each entry whose EN text exactly matches a key in mapping,
// the slot is replaced with the first corresponding Swedish value. If there are
// additional distinct Swedish values for the same English key, new entries are
// appended to the file — each a copy of the matching entry with a different EN
// slot — so the SE engine can find a match for every Swedish variant.
//
// Returns the number of entries updated (including appended entries) and any
// write error.
func Patch(path string, mapping map[string][][]byte) (int, error) {
	if len(mapping) == 0 {
		return 0, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading speech.info: %w", err)
	}

	count := 0

	// extra holds (full entry bytes) to append after all in-place patches.
	var extra [][]byte

	// Entry 0: no cue-name header, EN slot starts at entry0Base.
	// Entry 0 has no XACT cue so we only update in-place; no extras appended.
	if en0 := slotString(data[entry0Base : entry0Base+slotSize]); en0 != "" {
		if svList, ok := mapping[en0]; ok && len(svList) > 0 {
			writeSlot(data[entry0Base:entry0Base+slotSize], svList[0])
			count++
		}
	}

	// Entries 1+: each has a headerSize-byte cue header followed by language slots.
	nEntries := (len(data) - entry1Base) / entryStride
	for i := 0; i < nEntries; i++ {
		entryOff := entry1Base + i*entryStride
		enOffset := entryOff + headerSize
		en := slotString(data[enOffset : enOffset+slotSize])
		if en == "" {
			continue
		}
		svList, ok := mapping[en]
		if !ok || len(svList) == 0 {
			continue
		}
		writeSlot(data[enOffset:enOffset+slotSize], svList[0])
		count++

		// For each additional Swedish variant, clone the entry and update EN slot.
		for _, sv := range svList[1:] {
			clone := make([]byte, entryStride)
			copy(clone, data[entryOff:entryOff+entryStride])
			writeSlot(clone[headerSize:headerSize+slotSize], sv)
			extra = append(extra, clone)
			count++
		}
	}

	if count == 0 {
		return 0, nil
	}

	for _, e := range extra {
		data = append(data, e...)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return 0, fmt.Errorf("writing speech.info: %w", err)
	}
	return count, nil
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

// writeSlot writes text into a fixed-size slot.
// The slot is formatted as the original speech.info entries: text bytes,
// followed by a null terminator, followed by 0x20 (space) fill bytes.
// This matches the format the SE engine expects when building its lookup key.
func writeSlot(slot []byte, text []byte) {
	// Space-fill first, matching the original speech.info format.
	for i := range slot {
		slot[i] = 0x20
	}
	n := len(text)
	if n > len(slot)-1 {
		n = len(slot) - 1 // leave room for null terminator
	}
	copy(slot, text[:n])
	slot[n] = 0 // null terminator after text
}
