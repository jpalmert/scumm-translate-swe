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

// SPEECH-001: buildSpeechMapping builds EN→[]SCUMM_bytes from aligned files.
// Multi-page strings are split on \255\003 so each sentence maps individually.
func TestBuildSpeechMapping(t *testing.T) {
	en := []byte("[001:SCRP#0001] \n[001:SCRP#0001]The END for you!\n[001:SCRP#0002]Hello there.\\255\\003Goodbye now.\n")
	// Swedish data includes (opcode) prefix as produced by scummtr -A extraction.
	sv := []byte("[001:SCRP#0001](27) \n[001:SCRP#0001](27)SLUTET för dig!\n[001:SCRP#0002](27)Hej då.\\255\\003Adjö nu.\n")

	m := buildSpeechMapping(en, sv)

	// Single-page entry maps directly.
	sv1List, ok := m["The END for you!"]
	if !ok {
		t.Fatalf("expected 'The END for you!' in mapping, got keys: %v", mapKeys(m))
	}
	want := ScummBytes("SLUTET för dig!")
	if len(sv1List) != 1 || string(sv1List[0]) != string(want) {
		t.Errorf("SCUMM bytes: got %v, want [%v]", sv1List, want)
	}

	// Multi-page string: each sentence is mapped individually.
	sv2List, ok := m["Hello there."]
	if !ok {
		t.Fatalf("expected 'Hello there.' mapped as individual sentence")
	}
	if len(sv2List) != 1 || string(sv2List[0]) != string(ScummBytes("Hej då.")) {
		t.Errorf("first sentence: got %v, want [%v]", sv2List, ScummBytes("Hej då."))
	}
	sv3List, ok := m["Goodbye now."]
	if !ok {
		t.Fatalf("expected 'Goodbye now.' mapped as individual sentence")
	}
	if len(sv3List) != 1 || string(sv3List[0]) != string(ScummBytes("Adjö nu.")) {
		t.Errorf("second sentence: got %v, want [%v]", sv3List, ScummBytes("Adjö nu."))
	}

	// Full multi-page key must NOT be in the map (speech.info has individual sentences).
	if _, bad := m[`Hello there.\255\003Goodbye now.`]; bad {
		t.Errorf("full multi-page key should not be in mapping")
	}

	// Empty entries (just space or empty) should not appear as keys.
	if _, bad := m[" "]; bad {
		t.Errorf("empty-content entry should not be in mapping")
	}
}

// SPEECH-001b: buildSpeechMapping collects all distinct Swedish variants per EN key.
func TestBuildSpeechMappingMultipleVariants(t *testing.T) {
	// "Hello there." appears twice with different Swedish translations.
	en := []byte("[001:SCRP#0001]Hello there.\n[001:SCRP#0002]Hello there.\n[001:SCRP#0003]Hello there.\n")
	sv := []byte("[001:SCRP#0001]Hej där.\n[001:SCRP#0002]God dag.\n[001:SCRP#0003]Hej där.\n")

	m := buildSpeechMapping(en, sv)

	svList, ok := m["Hello there."]
	if !ok {
		t.Fatalf("expected 'Hello there.' in mapping")
	}
	// Two distinct Swedish translations: "Hej där." and "God dag." (third is duplicate of first).
	if len(svList) != 2 {
		t.Errorf("expected 2 distinct translations, got %d: %v", len(svList), svList)
	}
	if string(svList[0]) != string(ScummBytes("Hej där.")) {
		t.Errorf("first variant: got %v, want %v", svList[0], ScummBytes("Hej där."))
	}
	if string(svList[1]) != string(ScummBytes("God dag.")) {
		t.Errorf("second variant: got %v, want %v", svList[1], ScummBytes("God dag."))
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
	list, ok := m["Hello there."]
	if !ok || len(list) == 0 {
		t.Error("non-sword-fight entry should be included in mapping")
	}
}

func mapKeys(m map[string][][]byte) []string {
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
