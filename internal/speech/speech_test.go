package speech

import (
	"os"
	"path/filepath"
	"testing"
)

// buildTestSpeechInfo creates a minimal speech.info for testing.
// Layout: 16-byte file header, then entry0 (no cue header, 5 slots),
// then one cued entry (0x30-byte header + 5 slots).
func buildTestSpeechInfo(entry0EN, entry1CueName, entry1EN string) []byte {
	size := 0x10 + 5*slotSize + entryStride
	data := make([]byte, size)

	// File header (16 bytes): just magic-like placeholder
	data[0] = 0x01

	// Entry 0 EN slot at 0x10
	copy(data[entry0Base:], []byte(entry0EN))

	// Entry 1 cue name at entry1Base
	copy(data[entry1Base:], []byte(entry1CueName))

	// Entry 1 EN slot at entry1Base + headerSize
	copy(data[entry1Base+headerSize:], []byte(entry1EN))

	return data
}

// SPEECH-PATCH-001: Patch replaces matching EN slots.
func TestPatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speech.info")

	data := buildTestSpeechInfo("Hello world", "CUE_1_room_1_1", "I love a circus!")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	mapping := map[string][][]byte{
		"I love a circus!": {[]byte("Jag \x7clskar cirkus!")},
	}

	n, err := Patch(path, mapping)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 update, got %d", n)
	}

	// Read back and verify.
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	enSlot := slotString(updated[entry1Base+headerSize : entry1Base+headerSize+slotSize])
	if enSlot != "Jag \x7clskar cirkus!" {
		t.Errorf("EN slot after patch: %q", enSlot)
	}

	// Unmatched entry0 should be unchanged.
	e0Slot := slotString(updated[entry0Base : entry0Base+slotSize])
	if e0Slot != "Hello world" {
		t.Errorf("entry0 should be unchanged, got %q", e0Slot)
	}
}

// SPEECH-PATCH-002: Empty mapping → no write, returns 0.
func TestPatchEmptyMapping(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speech.info")

	data := buildTestSpeechInfo("Hello", "CUE", "Text")
	original := make([]byte, len(data))
	copy(original, data)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	n, err := Patch(path, nil)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 updates, got %d", n)
	}
}

// SPEECH-PATCH-003: writeSlot writes text + null terminator + space-fill.
func TestWriteSlot(t *testing.T) {
	slot := make([]byte, 10)
	for i := range slot {
		slot[i] = 0xFF // fill with non-zero
	}
	writeSlot(slot, []byte("Hi"))
	if slot[0] != 'H' || slot[1] != 'i' {
		t.Errorf("expected 'Hi', got %v", slot[:2])
	}
	if slot[2] != 0 {
		t.Errorf("slot[2] = %d, want 0 (null terminator)", slot[2])
	}
	for i := 3; i < 10; i++ {
		if slot[i] != 0x20 {
			t.Errorf("slot[%d] = %d, want 0x20 (space fill)", i, slot[i])
		}
	}
}

// SPEECH-PATCH-003b: writeSlot truncates text that exceeds slot capacity.
func TestWriteSlotTruncation(t *testing.T) {
	cases := []struct {
		name     string
		slotSize int
		text     string
		wantText string
	}{
		{"text exactly fits", 6, "Hello", "Hello"},
		{"text exceeds slot", 4, "Hello", "Hel"},
		{"empty text", 10, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			slot := make([]byte, tc.slotSize)
			writeSlot(slot, []byte(tc.text))
			got := slotString(slot)
			if got != tc.wantText {
				t.Errorf("writeSlot with %d-byte slot and %q: slotString = %q, want %q",
					tc.slotSize, tc.text, got, tc.wantText)
			}
			// Verify null terminator is present after text.
			nullPos := len(tc.wantText)
			if nullPos < tc.slotSize && slot[nullPos] != 0 {
				t.Errorf("expected null terminator at slot[%d], got %d", nullPos, slot[nullPos])
			}
		})
	}
}

// SPEECH-PATCH-003c: slotString returns entire slot when no null terminator is present.
func TestSlotStringNoNull(t *testing.T) {
	cases := []struct {
		name string
		slot []byte
		want string
	}{
		{"no null in slot", []byte{'H', 'e', 'l', 'l', 'o'}, "Hello"},
		{"null at start", []byte{0, 'a', 'b'}, ""},
		{"normal null-terminated", []byte{'H', 'i', 0, 0x20, 0x20}, "Hi"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := slotString(tc.slot)
			if got != tc.want {
				t.Errorf("slotString(%v) = %q, want %q", tc.slot, got, tc.want)
			}
		})
	}
}

// SPEECH-PATCH-004: Multiple Swedish variants append extra entries with same cue header.
func TestPatchAppendsExtraEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speech.info")

	cueName := "CUE_1_room_1_1"
	data := buildTestSpeechInfo("", cueName, "Hello there.")
	origLen := len(data)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	mapping := map[string][][]byte{
		"Hello there.": {[]byte("Hej där."), []byte("God dag.")},
	}

	n, err := Patch(path, mapping)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 (1 in-place + 1 appended), got %d", n)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// File should have grown by one entryStride.
	wantLen := origLen + entryStride
	if len(updated) != wantLen {
		t.Errorf("file size: got %d, want %d", len(updated), wantLen)
	}

	// Original entry: EN slot = first SV value.
	en1 := slotString(updated[entry1Base+headerSize : entry1Base+headerSize+slotSize])
	if en1 != "Hej där." {
		t.Errorf("original entry EN slot: got %q, want %q", en1, "Hej där.")
	}

	// Appended entry: same cue header, EN slot = second SV value.
	appended := entry1Base + entryStride
	cue2 := string(updated[appended : appended+len(cueName)])
	if cue2 != cueName {
		t.Errorf("appended entry cue: got %q, want %q", cue2, cueName)
	}
	en2 := slotString(updated[appended+headerSize : appended+headerSize+slotSize])
	if en2 != "God dag." {
		t.Errorf("appended entry EN slot: got %q, want %q", en2, "God dag.")
	}
}
