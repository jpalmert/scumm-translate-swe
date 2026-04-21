// Package uitext reads, writes, and patches Monkey Island Special Edition
// uiText.info files. These files contain UI/menu text, object names, verb
// labels, and other strings used by the SE renderer.
//
// Format (little-endian, Latin-1 encoding):
//
//	The file is a flat sequence of entries. Each entry consists of 6 fields
//	of 256 bytes each (1536 bytes total):
//	  Field 0: key   (ASCII identifier, e.g. "MENU_BACK")
//	  Field 1: EN    (English text)
//	  Field 2: FR    (French text)
//	  Field 3: IT    (Italian text)
//	  Field 4: DE    (German text)
//	  Field 5: ES    (Spanish text)
//
//	Each field is null-terminated and padded with spaces (0x20) to exactly
//	256 bytes. Maximum string length is 255 characters.
//	Text encoding is Latin-1 (ISO 8859-1 / Windows-1252 subset).
package uitext

import (
	"fmt"
	"unicode/utf8"
)

const (
	FieldSize   = 256
	FieldCount  = 6 // key + 5 languages
	EntrySize   = FieldCount * FieldSize
	LangEN      = 1 // index into the 6 fields
	MaxStrLen   = FieldSize - 1
)

// Entry represents one keyed text entry with translations in 5 languages.
type Entry struct {
	Key   string
	Texts [5]string // [0]=EN, [1]=FR, [2]=IT, [3]=DE, [4]=ES
}

// Read parses a uiText.info binary blob into entries.
func Read(data []byte) ([]Entry, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("uitext: empty data")
	}
	if len(data)%EntrySize != 0 {
		return nil, fmt.Errorf("uitext: data size %d is not a multiple of entry size %d", len(data), EntrySize)
	}

	n := len(data) / EntrySize
	entries := make([]Entry, n)
	for i := range entries {
		base := i * EntrySize
		entries[i].Key = decodeField(data[base : base+FieldSize])
		for lang := 0; lang < 5; lang++ {
			off := base + (lang+1)*FieldSize
			entries[i].Texts[lang] = decodeField(data[off : off+FieldSize])
		}
	}
	return entries, nil
}

// Write serialises entries back to uiText.info binary format.
func Write(entries []Entry) ([]byte, error) {
	out := make([]byte, len(entries)*EntrySize)
	for i, e := range entries {
		base := i * EntrySize
		if err := encodeField(out[base:base+FieldSize], e.Key); err != nil {
			return nil, fmt.Errorf("uitext: entry %d key %q: %w", i, e.Key, err)
		}
		for lang := 0; lang < 5; lang++ {
			off := base + (lang+1)*FieldSize
			if err := encodeField(out[off:off+FieldSize], e.Texts[lang]); err != nil {
				return nil, fmt.Errorf("uitext: entry %d key %q lang %d: %w", i, e.Key, lang, err)
			}
		}
	}
	return out, nil
}

// PatchEnglish replaces the English text slot for each key present in
// translations. Keys not found in the data are silently skipped. Returns
// the patched data (a modified copy of the input).
func PatchEnglish(data []byte, translations map[string]string) ([]byte, error) {
	if len(data)%EntrySize != 0 {
		return nil, fmt.Errorf("uitext: data size %d is not a multiple of entry size %d", len(data), EntrySize)
	}

	patched := make([]byte, len(data))
	copy(patched, data)

	n := len(patched) / EntrySize
	for i := 0; i < n; i++ {
		base := i * EntrySize
		key := decodeField(patched[base : base+FieldSize])
		swe, ok := translations[key]
		if !ok {
			continue
		}
		off := base + LangEN*FieldSize
		if err := encodeField(patched[off:off+FieldSize], swe); err != nil {
			return nil, fmt.Errorf("uitext: key %q: %w", key, err)
		}
	}
	return patched, nil
}

// decodeField reads a 256-byte slot as a null-terminated Latin-1 string,
// converting to UTF-8.
func decodeField(slot []byte) string {
	// Find null terminator.
	n := 0
	for n < len(slot) && slot[n] != 0 {
		n++
	}
	// Convert Latin-1 → UTF-8.
	runes := make([]rune, n)
	for i := 0; i < n; i++ {
		runes[i] = rune(slot[i])
	}
	return string(runes)
}

// encodeField writes a UTF-8 string into a 256-byte slot as null-terminated
// Latin-1, padded with spaces (0x20).
func encodeField(slot []byte, s string) error {
	latin1, err := utf8ToLatin1(s)
	if err != nil {
		return err
	}
	if len(latin1) > MaxStrLen {
		return fmt.Errorf("string too long (%d bytes, max %d)", len(latin1), MaxStrLen)
	}
	// Fill with spaces first, then write string + null.
	for i := range slot {
		slot[i] = 0x20
	}
	copy(slot, latin1)
	slot[len(latin1)] = 0
	return nil
}

// utf8ToLatin1 converts a UTF-8 string to Latin-1 bytes. Returns an error
// if any character is outside the Latin-1 range (U+0000–U+00FF).
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
