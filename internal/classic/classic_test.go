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
	want := "[001:OBNA#0016]djungel\n[001:VERB#0026]Det \x5cr en flaska.\n[002:SCRP#0037] \n"
	want = "[001:OBNA#0016]djungel\n[001:VERB#0026]Det \\124r en flaska.\n[002:SCRP#0037] \n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// SPEECH-001: buildSpeechMapping builds EN→SCUMM_bytes from aligned files.
func TestBuildSpeechMapping(t *testing.T) {
	en := []byte("[001:SCRP#0001] \n[001:SCRP#0001]The END for you!\n")
	// Swedish data includes (opcode) prefix as produced by scummtr -A extraction.
	sv := []byte("[001:SCRP#0001](27) \n[001:SCRP#0001](27)SLUTET för dig!\n")

	m := buildSpeechMapping(en, sv)

	// The translated entry should appear in the map.
	sv1, ok := m["The END for you!"]
	if !ok {
		t.Fatalf("expected 'The END for you!' in mapping, got keys: %v", mapKeys(m))
	}
	// ö → 0x7D, ä would be 0x7C — verify 'ö' in "för" maps correctly.
	want := ScummBytes("SLUTET för dig!")
	if string(sv1) != string(want) {
		t.Errorf("SCUMM bytes: got %v, want %v", sv1, want)
	}

	// Empty entries (just space or empty) should not appear as keys.
	if _, bad := m[" "]; bad {
		t.Errorf("empty-content entry should not be in mapping")
	}
}

// SPEECH-002: sword-fight insults and comebacks are excluded from the mapping.
func TestBuildSpeechMappingExcludesSwordFight(t *testing.T) {
	en := []byte("[088:SCRP#0085]You fight like a dairy farmer.\n[088:SCRP#0086]How appropriate.  You fight like a cow.\n[001:OBNA#0001]Hello there.\n")
	sv := []byte("[088:SCRP#0085](27)Du slåss som en bonde.\n[088:SCRP#0086](27)Lämpligt.  Du slåss som en ko.\n[001:OBNA#0001](__)Hej där.\n")

	m := buildSpeechMapping(en, sv)

	if _, ok := m["You fight like a dairy farmer."]; ok {
		t.Error("insult should be excluded from mapping")
	}
	if _, ok := m["How appropriate.  You fight like a cow."]; ok {
		t.Error("comeback should be excluded from mapping")
	}
	if _, ok := m["Hello there."]; !ok {
		t.Error("non-sword-fight entry should be included in mapping")
	}
}

func mapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SCUMM-BYTES-001: ScummBytes encodes Swedish UTF-8 to SCUMM byte values.
func TestScummBytes(t *testing.T) {
	cases := []struct {
		in   string
		want []byte
	}{
		{"Å", []byte{0x5B}},
		{"Ä", []byte{0x5C}},
		{"Ö", []byte{0x5D}},
		{"å", []byte{0x7B}},
		{"ä", []byte{0x7C}},
		{"ö", []byte{0x7D}},
		{"é", []byte{0x82}},
		{"Det här", []byte{'D', 'e', 't', ' ', 'h', 0x7C, 'r'}},
	}
	for _, tc := range cases {
		got := ScummBytes(tc.in)
		if string(got) != string(tc.want) {
			t.Errorf("ScummBytes(%q) = %v, want %v", tc.in, got, tc.want)
		}
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
