package classic

import (
	"os"
	"path/filepath"
	"strings"
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

// ENCODE-005: Whitespace-only content is also padded to a single space.
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

// MERGE-001: Swedish entries replace EN entries at matching resource+position.
func TestMergeTranslation(t *testing.T) {
	en := []byte("[001:OBNA#0001]jungle\n[001:OBNA#0002]beach\n[088:SCRP#0085]The END for you!\n[088:SCRP#0085]You fight like a cow.\n")
	sv := []byte("\n\n[088:SCRP#0085](27)SLUTET för dig!\n[088:SCRP#0085]Du slåss som en ko.\n")

	got := string(mergeTranslation(en, sv))

	// EN entries with no SV translation should be kept.
	if !strings.Contains(got, "[001:OBNA#0001]jungle") {
		t.Errorf("expected untranslated EN entry, got:\n%s", got)
	}
	// SV entries should replace corresponding EN entries; (27) prefix stripped.
	if !strings.Contains(got, "[088:SCRP#0085]SLUTET för dig!") {
		t.Errorf("expected Swedish entry without (27) prefix, got:\n%s", got)
	}
	if !strings.Contains(got, "[088:SCRP#0085]Du slåss som en ko.") {
		t.Errorf("expected second Swedish entry, got:\n%s", got)
	}
	// Original EN should not appear for translated entries.
	if strings.Contains(got, "The END for you!") {
		t.Errorf("original EN entry should be replaced, got:\n%s", got)
	}
}

// MERGE-002: stripParenPrefix removes (NN) prefixes and leaves other text alone.
func TestStripParenPrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"(27)text", "text"},
		{"(0)text", "text"},
		{"(123)Swedish text", "Swedish text"},
		{"text without prefix", "text without prefix"},
		{"(not digits)text", "(not digits)text"},
		{"", ""},
		{"(27)", ""},
	}
	for _, tc := range cases {
		got := stripParenPrefix(tc.in)
		if got != tc.want {
			t.Errorf("stripParenPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// SPEECH-001: buildSpeechMapping builds EN→SCUMM_bytes from aligned files.
func TestBuildSpeechMapping(t *testing.T) {
	en := []byte("[088:SCRP#0085] \n[088:SCRP#0085]The END for you!\n")
	sv := []byte("[088:SCRP#0085](27) \n[088:SCRP#0085](27)SLUTET för dig!\n")

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
