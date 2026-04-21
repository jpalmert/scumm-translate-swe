package hints

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// buildTestHints constructs a minimal hints binary with an index matrix
// at offset 0x76B0 and strings in the pool area after the matrix.
// groups is a slice of [5][<=4]string — 5 languages, up to 4 hints each.
func buildTestHints(t *testing.T, groups [][5][]string) []byte {
	t.Helper()

	numEntries := len(groups) * numLangs
	matrixSize := numEntries * matrixEntrySize

	// Fill everything before 0x76B0 with zeros (preamble).
	buf := make([]byte, indexMatrixOffset)

	// Build matrix entries and string pool.
	matrixBytes := make([]byte, matrixSize)
	poolOffset := indexMatrixOffset + matrixSize
	var pool []byte

	// Collect all strings in order with their entry/level coordinates.
	type stringRef struct {
		entryBase int
		level     int
		latin1    []byte
	}
	var allStrings []stringRef
	for g, group := range groups {
		for lang := 0; lang < numLangs; lang++ {
			entryIdx := g*numLangs + lang
			entryBase := entryIdx * matrixEntrySize
			hints := group[lang]
			for level := 0; level < maxHintsPerSeries; level++ {
				if level < len(hints) && hints[level] != "" {
					latin1, err := utf8ToLatin1(hints[level])
					if err != nil {
						t.Fatalf("utf8ToLatin1: %v", err)
					}
					allStrings = append(allStrings, stringRef{entryBase, level, latin1})
				}
			}
		}
	}
	for i, ref := range allStrings {
		stringAbsAddr := uint32(poolOffset) + uint32(len(pool))
		fieldAddr := indexMatrixOffset + uint32(ref.entryBase) + uint32(ref.level)*4
		relOff := stringAbsAddr - fieldAddr
		binary.LittleEndian.PutUint32(matrixBytes[ref.entryBase+ref.level*4:], relOff)

		pool = append(pool, ref.latin1...)
		pool = append(pool, 0)
		// Pad to 16-byte alignment (except last string — match real file behavior).
		if i < len(allStrings)-1 {
			for len(pool)%poolAlign != 0 {
				pool = append(pool, 0)
			}
		}
	}

	buf = append(buf, matrixBytes...)
	buf = append(buf, pool...)
	return buf
}

func TestParseSerialize_RoundTrip(t *testing.T) {
	groups := [][5][]string{
		{
			{"You should go to the SCUMM Bar."},
			{"Tu devrais aller au SCUMM Bar."},
			{"Du solltest der SCUMM-Bar einen Besuch abstatten."},
			{"Dovresti andare allo SCUMM Bar."},
			{"Debes ir al SCUMM Bar."},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out := h.Serialize()
	if !bytes.Equal(out, data) {
		t.Errorf("Serialize(Parse(data)) != data: got %d bytes, want %d", len(out), len(data))
	}
}

func TestExtractEnglish(t *testing.T) {
	groups := [][5][]string{
		{
			{"Hint one level one", "Hint one level two"},
			{"Indice un niveau un", "Indice un niveau deux"},
			{"Hinweis eins Stufe eins", "Hinweis eins Stufe zwei"},
			{"Suggerimento uno livello uno", "Suggerimento uno livello due"},
			{"Pista uno nivel uno", "Pista uno nivel dos"},
		},
		{
			{"Hint two single"},
			{"Indice deux simple"},
			{"Hinweis zwei einfach"},
			{"Suggerimento due semplice"},
			{"Pista dos simple esta"},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	if len(en) != 3 {
		t.Fatalf("got %d English strings, want 3", len(en))
	}
	if en[0].Text != "Hint one level one" {
		t.Errorf("en[0] = %q", en[0].Text)
	}
	if en[1].Text != "Hint one level two" {
		t.Errorf("en[1] = %q", en[1].Text)
	}
	if en[2].Text != "Hint two single" {
		t.Errorf("en[2] = %q", en[2].Text)
	}
}

func TestReplaceStrings_SameLength(t *testing.T) {
	groups := [][5][]string{
		{
			{"Go to the SCUMM Bar."},
			{"Va au SCUMM Bar ok."},
			{"Geh zur SCUMM-Bar!."},
			{"Vai allo SCUMM Bar."},
			{"Ve al SCUMM Bar ok."},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	replacements := map[uint32]string{
		en[0].Addr: "Gå till SCUMM Bar.",
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	after := h.ExtractEnglish()
	if after[0].Text != "Gå till SCUMM Bar." {
		t.Errorf("after = %q, want %q", after[0].Text, "Gå till SCUMM Bar.")
	}

	// Verify non-English untouched.
	h2, _ := Parse(h.Serialize())
	frAddr := h2.resolveStringAddr(1, 0)
	if frAddr == 0 {
		t.Fatal("French entry not found after rebuild")
	}
	frText := h2.readStringAt(frAddr)
	if frText != "Va au SCUMM Bar ok." {
		t.Errorf("French text changed: %q", frText)
	}
}

func TestReplaceStrings_LongerString(t *testing.T) {
	groups := [][5][]string{
		{
			{"Short."},
			{"Court."},
			{"Kurze."},
			{"Breve."},
			{"Corto."},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	originalAddr := en[0].Addr

	// Replace with something much longer than the original slot.
	longReplacement := "This is a much longer Swedish translation that exceeds the original slot size by a lot"
	replacements := map[uint32]string{
		originalAddr: longReplacement,
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	// Verify the replacement took effect.
	after := h.ExtractEnglish()
	if after[0].Text != longReplacement {
		t.Errorf("got %q, want %q", after[0].Text, longReplacement)
	}

	// Verify all 5 languages still resolve correctly.
	h2, _ := Parse(h.Serialize())
	langs := []string{"EN", "FR", "DE", "IT", "ES"}
	origTexts := []string{longReplacement, "Court.", "Kurze.", "Breve.", "Corto."}
	for langIdx := 0; langIdx < numLangs; langIdx++ {
		addr := h2.resolveStringAddr(langIdx, 0)
		if addr == 0 {
			t.Errorf("%s: entry not found", langs[langIdx])
			continue
		}
		text := h2.readStringAt(addr)
		if text != origTexts[langIdx] {
			t.Errorf("%s: got %q, want %q", langs[langIdx], text, origTexts[langIdx])
		}
	}
}

func TestReplaceStrings_ShorterString(t *testing.T) {
	groups := [][5][]string{
		{
			{"This is a long English hint string that takes up quite a bit of space."},
			{"Ceci est une longue phrase francaise pour les indices du jeu."},
			{"Dies ist ein langer deutscher Hinweistext fuer das Spiel ok."},
			{"Questo e un lungo suggerimento italiano per il gioco finale."},
			{"Esta es una larga pista en castellano para el juego completo."},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	replacements := map[uint32]string{
		en[0].Addr: "Kort.",
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	after := h.ExtractEnglish()
	if after[0].Text != "Kort." {
		t.Errorf("got %q", after[0].Text)
	}

	// File should be smaller now.
	if len(h.Serialize()) >= len(data) {
		t.Errorf("file did not shrink: was %d, now %d", len(data), len(h.Serialize()))
	}
}

func TestReplaceStrings_MultipleGroups(t *testing.T) {
	groups := [][5][]string{
		{
			{"Hint A level 1", "Hint A level 2"},
			{"Indice A niveau 1", "Indice A niveau 2"},
			{"Hinweis A Stufe 1!", "Hinweis A Stufe 2!"},
			{"Suggerimento A lv1", "Suggerimento A lv2"},
			{"Pista A nivel uno!", "Pista A nivel dos!"},
		},
		{
			{"Hint B single"},
			{"Indice B simple"},
			{"Hinweis B einzige"},
			{"Suggerimento B uno"},
			{"Pista B sola aqui"},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	if len(en) != 3 {
		t.Fatalf("got %d English, want 3", len(en))
	}

	// Replace all English strings with longer text.
	replacements := map[uint32]string{
		en[0].Addr: "Swedish hint A level 1 is now much longer than before yes indeed",
		en[1].Addr: "Swedish hint A level 2 is also much longer than the original text",
		en[2].Addr: "Swedish hint B is also longer than before",
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	// Verify all English strings.
	after := h.ExtractEnglish()
	if len(after) != 3 {
		t.Fatalf("after: got %d English, want 3", len(after))
	}
	for i, want := range []string{
		"Swedish hint A level 1 is now much longer than before yes indeed",
		"Swedish hint A level 2 is also much longer than the original text",
		"Swedish hint B is also longer than before",
	} {
		if after[i].Text != want {
			t.Errorf("en[%d] = %q, want %q", i, after[i].Text, want)
		}
	}

	// Verify non-English still intact.
	h2, _ := Parse(h.Serialize())
	// Group 0 French (entry 1), level 0
	frAddr := h2.resolveStringAddr(1, 0)
	if h2.readStringAt(frAddr) != "Indice A niveau 1" {
		t.Errorf("FR group 0 level 0 changed: %q", h2.readStringAt(frAddr))
	}
	// Group 1 German (entry 7), level 0
	deAddr := h2.resolveStringAddr(7, 0)
	if h2.readStringAt(deAddr) != "Hinweis B einzige" {
		t.Errorf("DE group 1 level 0 changed: %q", h2.readStringAt(deAddr))
	}
}

func TestReplaceStrings_SwedishChars(t *testing.T) {
	groups := [][5][]string{
		{
			{"Pick up the root beer."},
			{"Prends la biere de racine."},
			{"Nimm das Malzbier mit ok."},
			{"Prendi la bottiglia di orzata."},
			{"Recoge la cerveza de raices."},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	replacements := map[uint32]string{
		en[0].Addr: "Hämta ölet hos Stan. Åäö är bäst!",
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	after := h.ExtractEnglish()
	if after[0].Text != "Hämta ölet hos Stan. Åäö är bäst!" {
		t.Errorf("text = %q", after[0].Text)
	}

	// Verify Latin-1 encoding in raw bytes.
	raw := h.data[after[0].Addr:]
	if raw[1] != 0xE4 { // ä
		t.Errorf("byte 1 = 0x%02X, want 0xE4 (ä)", raw[1])
	}
	// "Hämta ölet..." — in Latin-1: H(48) ä(E4) m t a ' ' ö(F6) ...
	if raw[6] != 0xF6 { // ö is at Latin-1 byte 6
		t.Errorf("byte 6 = 0x%02X, want 0xF6 (ö)", raw[6])
	}
}

func TestReplaceStrings_InvalidAddr(t *testing.T) {
	groups := [][5][]string{
		{
			{"Hello there friend"},
			{"Bonjour mon ami ici"},
			{"Hallo mein Freund da"},
			{"Ciao il mio amico qui"},
			{"Hola mi amigo aqui"},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	replacements := map[uint32]string{
		9999999: "Bogus",
	}

	if err := h.ReplaceStrings(replacements); err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestReplaceStrings_NonLatin1(t *testing.T) {
	groups := [][5][]string{
		{
			{"Hello there friend"},
			{"Bonjour mon ami ici"},
			{"Hallo mein Freund da"},
			{"Ciao il mio amico qui"},
			{"Hola mi amigo aqui"},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	replacements := map[uint32]string{
		en[0].Addr: "Ā", // U+0100, outside Latin-1
	}

	if err := h.ReplaceStrings(replacements); err == nil {
		t.Error("expected error for non-Latin-1 character")
	}
}

func TestReplaceStrings_NoReplacements(t *testing.T) {
	groups := [][5][]string{
		{
			{"Hello there friend"},
			{"Bonjour mon ami ici"},
			{"Hallo mein Freund da"},
			{"Ciao il mio amico qui"},
			{"Hola mi amigo aqui"},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Empty replacements should still produce identical output.
	if err := h.ReplaceStrings(map[uint32]string{}); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	if !bytes.Equal(h.Serialize(), data) {
		t.Error("empty replacement changed the file")
	}
}

func TestReplaceStrings_VeryLongString(t *testing.T) {
	groups := [][5][]string{
		{
			{"X"},
			{"Y"},
			{"Z"},
			{"W"},
			{"Q"},
		},
	}
	data := buildTestHints(t, groups)

	h, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	en := h.ExtractEnglish()
	// Replace a 1-char string with a 500-char string.
	long := strings.Repeat("A", 500)
	replacements := map[uint32]string{
		en[0].Addr: long,
	}

	if err := h.ReplaceStrings(replacements); err != nil {
		t.Fatalf("ReplaceStrings: %v", err)
	}

	after := h.ExtractEnglish()
	if after[0].Text != long {
		t.Errorf("got length %d, want 500", len(after[0].Text))
	}
}

func TestParse_TooShort(t *testing.T) {
	if _, err := Parse(make([]byte, 100)); err == nil {
		t.Error("expected error for data too short")
	}
}

func TestParse_ZeroEntries(t *testing.T) {
	data := make([]byte, indexMatrixOffset+4)
	if _, err := Parse(data); err == nil {
		t.Error("expected error for 0 entries")
	}
}
