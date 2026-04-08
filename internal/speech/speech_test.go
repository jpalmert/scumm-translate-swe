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

	mapping := map[string][]byte{
		"I love a circus!": []byte("Jag \x7clskar cirkus!"),
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

// SPEECH-PATCH-003: writeSlot zero-pads remaining bytes.
func TestWriteSlot(t *testing.T) {
	slot := make([]byte, 10)
	for i := range slot {
		slot[i] = 0xFF // fill with non-zero
	}
	writeSlot(slot, []byte("Hi"))
	if slot[0] != 'H' || slot[1] != 'i' {
		t.Errorf("expected 'Hi', got %v", slot[:2])
	}
	for i := 2; i < 10; i++ {
		if slot[i] != 0 {
			t.Errorf("slot[%d] = %d, want 0", i, slot[i])
		}
	}
}
