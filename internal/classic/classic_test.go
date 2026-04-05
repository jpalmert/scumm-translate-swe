package classic

import (
	"os"
	"path/filepath"
	"testing"
)

// ENCODE-001: Swedish Windows-1252 bytes are replaced with SCUMM escape codes.
func TestEncodeForScummtr(t *testing.T) {
	// Each Swedish char must map to the correct \NNN escape code.
	cases := []struct {
		in  byte
		out string
	}{
		{0xC5, `\091`}, // Å
		{0xC4, `\092`}, // Ä
		{0xD6, `\093`}, // Ö
		{0xE5, `\123`}, // å
		{0xE4, `\124`}, // ä
		{0xF6, `\125`}, // ö
		{0xE9, `\130`}, // é
	}

	for _, tc := range cases {
		dir := t.TempDir()
		p := filepath.Join(dir, "t.txt")
		os.WriteFile(p, []byte{tc.in}, 0644)

		got, err := encodeForScummtr(p)
		if err != nil {
			t.Fatalf("0x%02X: %v", tc.in, err)
		}
		if string(got) != tc.out {
			t.Errorf("0x%02X: got %q, want %q", tc.in, got, tc.out)
		}
	}
}

// ENCODE-002: ASCII bytes are passed through unchanged.
func TestEncodeForScummtrASCII(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.txt")
	input := "Hello, world!\n"
	os.WriteFile(p, []byte(input), 0644)

	got, err := encodeForScummtr(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

// ENCODE-003: Mixed content — Swedish chars encoded, rest unchanged.
func TestEncodeForScummtrMixed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.txt")
	// "Jag är glad" in Windows-1252 (ä = 0xE4)
	input := []byte("Jag \xe4r glad")
	os.WriteFile(p, input, 0644)

	got, err := encodeForScummtr(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `Jag \124r glad`
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
