package uitext

import (
	"bytes"
	"strings"
	"testing"
)

// buildField creates a 256-byte field from a string: null-terminated, space-padded.
func buildField(s string) []byte {
	b := make([]byte, FieldSize)
	for i := range b {
		b[i] = 0x20 // space padding
	}
	latin1, _ := utf8ToLatin1(s)
	copy(b, latin1)
	b[len(latin1)] = 0
	return b
}

// buildEntry creates a 1536-byte entry from a key and 5 language strings.
func buildEntry(key string, texts [5]string) []byte {
	var buf []byte
	buf = append(buf, buildField(key)...)
	for _, t := range texts {
		buf = append(buf, buildField(t)...)
	}
	return buf
}

func TestReadWrite_RoundTrip(t *testing.T) {
	data := buildEntry("TEST_KEY", [5]string{"English", "Français", "Italiano", "Deutsch", "Español"})
	data = append(data, buildEntry("ANOTHER", [5]string{"Hello", "Bonjour", "Ciao", "Hallo", "Hola"})...)

	entries, err := Read(data)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Key != "TEST_KEY" {
		t.Errorf("key = %q, want TEST_KEY", entries[0].Key)
	}
	if entries[0].Texts[0] != "English" {
		t.Errorf("EN = %q, want English", entries[0].Texts[0])
	}
	if entries[1].Texts[1] != "Bonjour" {
		t.Errorf("FR = %q, want Bonjour", entries[1].Texts[1])
	}

	out, err := Write(entries)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Error("round-trip Write(Read(data)) != data")
	}
}

func TestReadWrite_SwedishChars(t *testing.T) {
	data := buildEntry("SWE", [5]string{"Åäö ÄÖ é", "", "", "", ""})

	entries, err := Read(data)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if entries[0].Texts[0] != "Åäö ÄÖ é" {
		t.Errorf("Swedish text = %q, want %q", entries[0].Texts[0], "Åäö ÄÖ é")
	}

	out, err := Write(entries)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Error("Swedish chars round-trip failed")
	}
}

func TestRead_EmptyData(t *testing.T) {
	if _, err := Read(nil); err == nil {
		t.Error("expected error for nil data")
	}
	if _, err := Read([]byte{}); err == nil {
		t.Error("expected error for empty data")
	}
}

func TestRead_WrongSize(t *testing.T) {
	if _, err := Read(make([]byte, 100)); err == nil {
		t.Error("expected error for data not a multiple of entry size")
	}
}

func TestPatchEnglish_Basic(t *testing.T) {
	data := buildEntry("MENU_BACK", [5]string{"Back", "Retour", "Indietro", "Zurück", "Volver"})
	data = append(data, buildEntry("MENU_YES", [5]string{"Yes", "Oui", "Sì", "Ja", "Sí"})...)

	translations := map[string]string{
		"MENU_BACK": "Tillbaka",
		"MENU_YES":  "Ja",
	}

	patched, err := PatchEnglish(data, translations)
	if err != nil {
		t.Fatalf("PatchEnglish: %v", err)
	}

	entries, err := Read(patched)
	if err != nil {
		t.Fatalf("Read patched: %v", err)
	}

	// EN slots should be replaced.
	if entries[0].Texts[0] != "Tillbaka" {
		t.Errorf("MENU_BACK EN = %q, want Tillbaka", entries[0].Texts[0])
	}
	if entries[1].Texts[0] != "Ja" {
		t.Errorf("MENU_YES EN = %q, want Ja", entries[1].Texts[0])
	}

	// Other language slots must be untouched.
	if entries[0].Texts[1] != "Retour" {
		t.Errorf("MENU_BACK FR = %q, want Retour", entries[0].Texts[1])
	}
	if entries[0].Texts[3] != "Zurück" {
		t.Errorf("MENU_BACK DE = %q, want Zurück", entries[0].Texts[3])
	}

	// Key must be untouched.
	if entries[0].Key != "MENU_BACK" {
		t.Errorf("key = %q, want MENU_BACK", entries[0].Key)
	}
}

func TestPatchEnglish_UnknownKeySkipped(t *testing.T) {
	data := buildEntry("MENU_BACK", [5]string{"Back", "Retour", "Indietro", "Zurück", "Volver"})

	translations := map[string]string{
		"NONEXISTENT_KEY": "Saknas",
	}

	patched, err := PatchEnglish(data, translations)
	if err != nil {
		t.Fatalf("PatchEnglish: %v", err)
	}

	entries, _ := Read(patched)
	if entries[0].Texts[0] != "Back" {
		t.Errorf("EN should be unchanged, got %q", entries[0].Texts[0])
	}
}

func TestPatchEnglish_SwedishChars(t *testing.T) {
	data := buildEntry("MENU_BACK", [5]string{"Back", "", "", "", ""})

	translations := map[string]string{
		"MENU_BACK": "Åäö är bäst",
	}

	patched, err := PatchEnglish(data, translations)
	if err != nil {
		t.Fatalf("PatchEnglish: %v", err)
	}

	entries, _ := Read(patched)
	if entries[0].Texts[0] != "Åäö är bäst" {
		t.Errorf("Swedish text = %q, want %q", entries[0].Texts[0], "Åäö är bäst")
	}

	// Verify Latin-1 encoding in raw bytes: å = 0xE5, ä = 0xE4, ö = 0xF6
	enSlot := patched[FieldSize : 2*FieldSize]
	if enSlot[0] != 0xC5 { // Å
		t.Errorf("byte 0 = 0x%02X, want 0xC5 (Å)", enSlot[0])
	}
	if enSlot[1] != 0xE4 { // ä
		t.Errorf("byte 1 = 0x%02X, want 0xE4 (ä)", enSlot[1])
	}
	if enSlot[2] != 0xF6 { // ö
		t.Errorf("byte 2 = 0x%02X, want 0xF6 (ö)", enSlot[2])
	}
}

func TestPatchEnglish_TooLong(t *testing.T) {
	data := buildEntry("KEY", [5]string{"Short", "", "", "", ""})

	long := strings.Repeat("X", 256) // exceeds 255
	translations := map[string]string{"KEY": long}

	if _, err := PatchEnglish(data, translations); err == nil {
		t.Error("expected error for string exceeding 255 chars")
	}
}

func TestPatchEnglish_WrongDataSize(t *testing.T) {
	if _, err := PatchEnglish(make([]byte, 100), nil); err == nil {
		t.Error("expected error for wrong data size")
	}
}

func TestWrite_TooLong(t *testing.T) {
	entries := []Entry{{Key: "K", Texts: [5]string{strings.Repeat("X", 256)}}}
	if _, err := Write(entries); err == nil {
		t.Error("expected error for string exceeding 255 chars")
	}
}

func TestUTF8ToLatin1_OutOfRange(t *testing.T) {
	// A character outside Latin-1 (e.g. U+0100 Ā) should fail.
	if _, err := utf8ToLatin1("Ā"); err == nil {
		t.Error("expected error for character outside Latin-1 range")
	}
}
