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
		{"Å", `\091`},
		{"Ä", `\092`},
		{"Ö", `\093`},
		{"å", `\123`},
		{"ä", `\124`},
		{"ö", `\125`},
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

// ENCODE-004: Empty-content lines get a space injected so scummtr accepts them
// while preserving their position (sequential matching within a resource).
func TestEncodeForScummtrPadsEmptyContentLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.txt")
	// Two strings in SCRP#0037: empty first, "gam" second. Both must be preserved.
	input := "[001:OBNA#0016]djungel\n[002:SCRP#0037]\n[002:SCRP#0037]gam\n[002:SCRP#0038]\n"
	os.WriteFile(p, []byte(input), 0644)

	got, err := encodeForScummtr(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "[001:OBNA#0016]djungel\n[002:SCRP#0037] \n[002:SCRP#0037]gam\n[002:SCRP#0038] \n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ENCODE-005: Whitespace-only content is preserved as-is (only truly empty content is padded).
func TestEncodeForScummtrPadsWhitespaceContentLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.txt")
	input := "[001:OBNA#0016]djungel\n[002:SCRP#0035] \n[002:SCRP#0036]strand\n"
	os.WriteFile(p, []byte(input), 0644)

	got, err := encodeForScummtr(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "[001:OBNA#0016]djungel\n[002:SCRP#0035] \n[002:SCRP#0036]strand\n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ENCODE-006: (opcode) prefixes are stripped from text before injection.
func TestEncodeForScummtrStripsOpcode(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.txt")
	// Swedish.txt format produced by scummtr -A extraction includes opcode prefixes.
	input := "[001:OBNA#0016](__)djungel\n[001:VERB#0026](D8)Det är en flaska.\n[002:SCRP#0037](93)\n"
	os.WriteFile(p, []byte(input), 0644)

	got, err := encodeForScummtr(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Opcodes stripped; empty content padded; Swedish chars encoded.
	want := "[001:OBNA#0016]djungel\n[001:VERB#0026]Det \\124r en flaska.\n[002:SCRP#0037] \n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// DECODE-001: DecodeScummtrEscapes converts scummtr escape sequences to raw bytes.
func TestDecodeScummtrEscapes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []byte
	}{
		{"plain ASCII", "hello", []byte("hello")},
		{"single escape", `\091`, []byte{91}},
		{"multiple escapes", `\091\092`, []byte{91, 92}},
		{"mixed text and escapes", `abc\091def`, []byte{'a', 'b', 'c', 91, 'd', 'e', 'f'}},
		{"double backslash", `\\`, []byte{0x5C}},
		{"trailing backslash", `hello\`, []byte{'h', 'e', 'l', 'l', 'o', '\\'}},
		{"empty string", "", []byte{}},
		{"escape value 255", `\255`, []byte{255}},
		{"escape value 000", `\000`, []byte{0}},
		{"backslash then non-digit", `\abc`, []byte{'\\', 'a', 'b', 'c'}},
		{"only two digits after backslash", `\09x`, []byte{'\\', '0', '9', 'x'}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DecodeScummtrEscapes(tc.in)
			if string(got) != string(tc.want) {
				t.Errorf("DecodeScummtrEscapes(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
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
	want := `Jag \124r glad`
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
