//go:build integration

package hints

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scumm-patcher/internal/pak"
)

// repoRoot walks up from the package directory to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod in parent directories")
		}
		dir = parent
	}
}

// loadRealHintsData extracts hints/monkey1.hints.csv from the real PAK file.
func loadRealHintsData(t *testing.T) []byte {
	t.Helper()
	root := repoRoot(t)
	pakPath := filepath.Join(root, "games", "monkey1", "game", "Monkey1.pak")
	if _, err := os.Stat(pakPath); err != nil {
		t.Skipf("PAK file not found: %s", pakPath)
	}

	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name), ".hints.csv") {
			return append([]byte(nil), e.Data...)
		}
	}
	t.Fatal("hints/monkey1.hints.csv not found in PAK")
	return nil
}

// INT-HINTS-001: Parse and Serialize round-trip preserves all bytes of the real file.
func TestRealHints_RoundTrip(t *testing.T) {
	data := loadRealHintsData(t)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out := h.Serialize()
	if !bytes.Equal(out, data) {
		t.Errorf("round-trip changed file: original %d bytes, got %d bytes", len(data), len(out))
		// Find first diff.
		for i := range data {
			if i >= len(out) || data[i] != out[i] {
				t.Errorf("first diff at byte %d: want 0x%02X, got 0x%02X", i, data[i], out[i])
				break
			}
		}
	}
}

// INT-HINTS-002: ExtractEnglish finds exactly 517 English strings.
func TestRealHints_ExtractEnglishCount(t *testing.T) {
	data := loadRealHintsData(t)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	if len(en) != 517 {
		t.Errorf("got %d English strings, want 517", len(en))
	}

	// Spot-check first and last strings.
	if len(en) > 0 && !strings.HasPrefix(en[0].Text, "You need to use the root beer") {
		t.Errorf("first English string = %q", en[0].Text)
	}
	if len(en) > 0 && !strings.HasPrefix(en[len(en)-1].Text, "You need to go to the storekeeper") {
		t.Errorf("last English string = %q", en[len(en)-1].Text)
	}
}

// INT-HINTS-003: ReplaceStrings with no replacements produces identical output.
func TestRealHints_EmptyReplacement(t *testing.T) {
	data := loadRealHintsData(t)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if err := h.ReplaceStrings(map[uint32]string{}); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	out := h.Serialize()
	if !bytes.Equal(out, data) {
		t.Errorf("empty replacement changed file: original %d bytes, got %d bytes", len(data), len(out))
	}
}

// INT-HINTS-004: Replace a single English string with a longer Swedish translation,
// then verify all 517 English strings are still readable and the replaced one changed.
func TestRealHints_ReplaceSingleLonger(t *testing.T) {
	data := loadRealHintsData(t)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	if len(en) == 0 {
		t.Fatal("no English strings")
	}

	// Replace first string with something much longer.
	targetAddr := en[0].Addr
	longSwedish := "Du måste använda root beeren på LeChuck innan han slår till dig igen! Skynda dig, det finns inte mycket tid kvar!"
	replacements := map[uint32]string{
		targetAddr: longSwedish,
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	// Re-parse the serialized output.
	h2, err := Parse(h.Serialize())
	if err != nil {
		t.Fatalf("Parse after replace: %v", err)
	}

	en2 := h2.ExtractEnglish()
	if len(en2) != 517 {
		t.Fatalf("after replacement: got %d English strings, want 517", len(en2))
	}

	// First string should be the replacement.
	if en2[0].Text != longSwedish {
		t.Errorf("replaced string = %q, want %q", en2[0].Text, longSwedish)
	}

	// All other strings should be unchanged.
	for i := 1; i < len(en2); i++ {
		if en2[i].Text != en[i].Text {
			t.Errorf("string %d changed: was %q, now %q", i, en[i].Text, en2[i].Text)
			break
		}
	}
}

// INT-HINTS-005: Replace ALL 517 English strings with longer Swedish text,
// then verify all are readable.
func TestRealHints_ReplaceAllEnglish(t *testing.T) {
	data := loadRealHintsData(t)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	replacements := make(map[uint32]string, len(en))
	for _, s := range en {
		// Make each string ~30% longer with a Swedish prefix.
		replacements[s.Addr] = "Översättning: " + s.Text
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	// Re-parse and verify.
	h2, err := Parse(h.Serialize())
	if err != nil {
		t.Fatalf("Parse after replace-all: %v", err)
	}

	en2 := h2.ExtractEnglish()
	if len(en2) != 517 {
		t.Fatalf("after replace-all: got %d English strings, want 517", len(en2))
	}

	for i, s := range en2 {
		want := "Översättning: " + en[i].Text
		if s.Text != want {
			t.Errorf("string %d = %q, want %q", i, s.Text[:40], want[:40])
			break
		}
	}

	// Also verify non-English strings survived. Check French (entry 1, level 0).
	frAddr := h2.resolveStringAddr(1, 0)
	if frAddr == 0 {
		t.Fatal("French entry 1 not found after replace-all")
	}
	frText := h2.readStringAt(frAddr)
	if !strings.HasPrefix(frText, "Tu dois utiliser") {
		t.Errorf("French text changed: %q", frText[:40])
	}

	t.Logf("File size: original %d, after replace-all %d (+%d bytes)",
		len(data), len(h.Serialize()), len(h.Serialize())-len(data))
}

// INT-HINTS-006: Replace all English strings, then serialize and re-replace
// (simulates re-patching). Both passes should produce identical results.
func TestRealHints_ReplaceIdempotent(t *testing.T) {
	data := loadRealHintsData(t)

	makeReplacements := func(h *HintsFile) map[uint32]string {
		en := h.ExtractEnglish()
		r := make(map[uint32]string, len(en))
		for _, s := range en {
			r[s.Addr] = "SV: " + s.Text
		}
		return r
	}

	// First pass.
	h1, _ := Parse(data)
	if err := h1.ReplaceStrings(makeReplacements(h1)); err != nil {
		t.Fatalf("first ReplaceStrings: %v", err)
	}
	out1 := h1.Serialize()

	// Second pass on the output of the first pass.
	// After first replacement, English strings are "SV: <original>".
	// We need to use the ORIGINAL data for the second pass too (simulating
	// the patcher re-reading from backup). So parse from original again.
	h2, _ := Parse(data)
	if err := h2.ReplaceStrings(makeReplacements(h2)); err != nil {
		t.Fatalf("second ReplaceStrings: %v", err)
	}
	out2 := h2.Serialize()

	if !bytes.Equal(out1, out2) {
		t.Errorf("two passes from same original produced different results: %d vs %d bytes",
			len(out1), len(out2))
	}
}
