package classic

import (
	"os"
	"path/filepath"
	"testing"
)

// ENCODE-001: Swedish UTF-8 characters are replaced with SCUMM escape codes.
func TestEncodeForScummtr(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"Å", `\197`},
		{"Ä", `\196`},
		{"Ö", `\214`},
		{"å", `\229`},
		{"ä", `\228`},
		{"ö", `\246`},
		{"é", `\130`},
	}

	for _, tc := range cases {
		dir := t.TempDir()
		p := filepath.Join(dir, "t.txt")
		os.WriteFile(p, []byte(tc.in), 0644)

		got, err := encodeForScummtr(p)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if string(got) != tc.out {
			t.Errorf("%q: got %q, want %q", tc.in, got, tc.out)
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
	os.WriteFile(p, []byte("Jag är glad"), 0644) // UTF-8

	got, err := encodeForScummtr(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `Jag \228r glad`
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
