// Package classic provides scummtr-based text injection for classic SCUMM game files.
//
// The embedded scummtr binary is extracted to a temporary directory at run time
// and invoked with the appropriate flags for Monkey Island 1.
package classic

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// scummCharMap maps UTF-8 Swedish characters to their SCUMM internal escape
// codes for Monkey Island 1. scummtr does not reliably auto-convert these via
// -c for the monkeycdalt game ID, so we pre-encode them before injection
// (matching the monkeycd_swe project approach).
//
// These custom SCUMM codes (91–93, 123–125, 130) replace ASCII punctuation that
// never appears in game dialog: '[', '\', ']' and '{', '|', '}'. The CHAR blocks
// are patched to put Swedish glyphs at those positions; the SE .font lookup
// table is also patched so the SE new-graphics renderer can find them.
var scummCharMap = []struct {
	from string
	to   string
}{
	{"Å", `\091`},
	{"Ä", `\092`},
	{"Ö", `\093`},
	{"å", `\123`},
	{"ä", `\124`},
	{"ö", `\125`},
	{"é", `\130`},
	{"ê", `\136`},
	{"®", `\015`}, // SCUMM stores ® as byte 0x0F; without this, UTF-8 0xC2 0xAE leaks in
}

// scummByteMap maps UTF-8 Swedish characters to their SCUMM byte values.
// Used for direct byte-level encoding (e.g. speech.info patching).
var scummByteMap = map[rune]byte{
	'Å': 0x5B,
	'Ä': 0x5C,
	'Ö': 0x5D,
	'å': 0x7B,
	'ä': 0x7C,
	'ö': 0x7D,
	'é': 0x82,
	'ê': 0x88,
	'®': 0x0F,
}

// ScummBytes converts UTF-8 text to the byte representation that scummtr
// injects into MONKEY1.001. Swedish characters are converted to their SCUMM
// byte codes (matching scummCharMap). Other non-ASCII characters pass through
// as their raw UTF-8 bytes, matching scummtr's behaviour of injecting them
// verbatim.  Used for building speech.info EN slot content.
func ScummBytes(text string) []byte {
	var b []byte
	for _, r := range text {
		if mapped, ok := scummByteMap[r]; ok {
			b = append(b, mapped)
		} else if r < 0x80 {
			b = append(b, byte(r))
		} else {
			// Pass through UTF-8 bytes for non-mapped non-ASCII runes (e.g. ®).
			// encodeBytes does not convert these, so scummtr injects their UTF-8
			// byte sequence verbatim.
			b = append(b, []byte(string(r))...)
		}
	}
	return b
}

// encodeForScummtr reads a UTF-8 encoded translation file and returns a copy
// with Swedish characters replaced by their SCUMM escape codes, and
// empty-content header lines padded with a single space so scummtr accepts them.
func encodeForScummtr(translationPath string) ([]byte, error) {
	data, err := os.ReadFile(translationPath)
	if err != nil {
		return nil, err
	}
	return encodeBytes(data), nil
}

// encodeBytes converts UTF-8 Swedish characters in scummtr-format data to
// SCUMM escape codes, strips leading (opcode) prefixes from the text portion
// of each header line, and pads empty-content header lines with a single space
// so scummtr does not reject them.
//
// The (opcode) prefix — e.g. "(D8)", "(__)" — is produced by scummtr -A
// extraction for identification purposes but must not appear in injected text.
func encodeBytes(data []byte) []byte {
	s := string(data)
	for _, m := range scummCharMap {
		s = strings.ReplaceAll(s, m.from, m.to)
	}
	// scummtr forbids empty lines, but empty-content entries must be preserved
	// in order: [room:TYPE#resnum] entries within the same resource are matched
	// positionally, so dropping one would shift all subsequent strings.
	// Replace empty/whitespace-only content with a single space so scummtr
	// accepts the line and injects a harmless space into an otherwise-unused string.
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if j := strings.IndexByte(line, ']'); j >= 0 && strings.HasPrefix(line, "[") {
			text := line[j+1:]
			text = stripOpcode(text)
			if text == "" {
				text = " "
			}
			lines[i] = line[:j+1] + text
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// stripOpcode removes a leading "(XX)" opcode prefix from a scummtr text field.
// scummtr -A extraction includes these for identification, but they must not
// appear in injected text.
func stripOpcode(text string) string {
	if len(text) > 0 && text[0] == '(' {
		if end := strings.IndexByte(text, ')'); end > 0 {
			return text[end+1:]
		}
	}
	return text
}

// ExtractLines extracts all dialog strings from the classic game files in
// gameDir and returns a slice of (header, text) pairs in document order.
// Both header (e.g. "[007:VERB#0075]") and text are plain strings without
// scummtr escape decoding — text may contain \NNN decimal escape sequences.
func ExtractLines(gameDir string) ([][2]string, error) {
	scummtrPath, tmpDir, cleanup, err := setupScummtr()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	outPath := filepath.Join(tmpDir, "extract.txt")
	if err := runScummtrExtract(scummtrPath, gameDir, outPath); err != nil {
		return nil, fmt.Errorf("extracting lines: %w", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, err
	}

	var lines [][2]string
	for _, line := range strings.Split(string(data), "\n") {
		j := strings.IndexByte(line, ']')
		if j < 0 || !strings.HasPrefix(line, "[") {
			continue
		}
		lines = append(lines, [2]string{line[:j+1], line[j+1:]})
	}
	return lines, nil
}

// DecodeScummtrEscapes converts scummtr escape sequences in s to raw bytes.
// scummtr uses two formats:
//   - \NNN (exactly three decimal digits) → the byte whose value is NNN
//   - \\ → a literal backslash byte (0x5C, which is the SCUMM code for 'Ä')
//
// Non-escaped bytes are copied as-is.
// Used to compare scummtr extract output against raw byte content (e.g. speech.info slots).
func DecodeScummtrEscapes(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if i+4 <= len(s) && s[i] == '\\' &&
			s[i+1] >= '0' && s[i+1] <= '9' &&
			s[i+2] >= '0' && s[i+2] <= '9' &&
			s[i+3] >= '0' && s[i+3] <= '9' {
			// \NNN → byte with decimal value NNN
			n := int(s[i+1]-'0')*100 + int(s[i+2]-'0')*10 + int(s[i+3]-'0')
			out = append(out, byte(n))
			i += 4
		} else if i+2 <= len(s) && s[i] == '\\' && s[i+1] == '\\' {
			// \\ → literal backslash (byte 0x5C = SCUMM code for 'Ä')
			out = append(out, 0x5C)
			i += 2
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return out
}

// BuildSpeechMapping extracts the original English text from the classic game
// files in gameDir, then parses the Swedish translation at translationPath, and
// returns a map from each original English string to all distinct SCUMM-encoded
// Swedish byte representations that correspond to it.
//
// A single English string can have multiple distinct Swedish translations when
// the same phrase appears in different contexts (e.g. "Hello" → ["Hej!", "Hallå!"]).
// Each translation produces a separate speech.info entry so the SE engine can
// match voiced lines to audio cues regardless of which Swedish variant appears.
func BuildSpeechMapping(gameDir, translationPath string) (map[string][][]byte, error) {
	scummtrPath, tmpDir, cleanup, err := setupScummtr()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	enPath := filepath.Join(tmpDir, "monkey1_en.txt")
	if err := runScummtrExtract(scummtrPath, gameDir, enPath); err != nil {
		return nil, fmt.Errorf("extracting source text: %w", err)
	}

	enData, err := os.ReadFile(enPath)
	if err != nil {
		return nil, err
	}
	svData, err := os.ReadFile(translationPath)
	if err != nil {
		return nil, err
	}

	return buildSpeechMapping(enData, svData), nil
}

// appendDistinct appends sv to list if it is not already present.
func appendDistinct(list [][]byte, sv []byte) [][]byte {
	for _, existing := range list {
		if string(existing) == string(sv) {
			return list
		}
	}
	return append(list, sv)
}

// swordFightResources lists the resource headers whose strings are sword-fight
// insults and comebacks. These are intentionally excluded from the speech
// mapping because the Swedish translation uses non-literal creative rewrites;
// matching the old English audio to the new Swedish text would be misleading.
var swordFightResources = map[string]bool{
	"[088:SCRP#0085]": true, // insults
	"[088:SCRP#0086]": true, // comebacks
}

// buildSpeechMapping builds the EN→[]SCUMM_bytes mapping from raw data slices.
// Both files use scummtr header format: "[room:TYPE#resnum]text".
// Entries are matched positionally within each resource.
// Sword-fight insult/comeback resources are excluded (see swordFightResources).
//
// Each English key maps to a list of all distinct Swedish byte representations
// that correspond to it. Duplicates (same SV bytes seen again) are not added.
func buildSpeechMapping(enData, svData []byte) map[string][][]byte {
	// Build SV groups: resource_header -> []text in order.
	// Strip any (opcode) prefix from text — scummtr -A extraction includes
	// these but they are not part of the translatable string.
	svGroups := make(map[string][]string)
	for _, line := range strings.Split(string(svData), "\n") {
		j := strings.IndexByte(line, ']')
		if j < 0 || !strings.HasPrefix(line, "[") {
			continue
		}
		svGroups[line[:j+1]] = append(svGroups[line[:j+1]], stripOpcode(line[j+1:]))
	}

	svPos := make(map[string]int)
	mapping := make(map[string][][]byte)
	for _, line := range strings.Split(string(enData), "\n") {
		j := strings.IndexByte(line, ']')
		if j < 0 || !strings.HasPrefix(line, "[") {
			continue
		}
		header := line[:j+1]
		enText := line[j+1:]
		p := svPos[header]
		svPos[header]++

		if swordFightResources[header] {
			continue
		}

		if svTexts, ok := svGroups[header]; ok && p < len(svTexts) {
			svText := svTexts[p]
			// speech.info stores individual sentences, not full multi-page strings.
			// Split both EN and SV on the page-break sequence and map each pair.
			//
			// EN text comes from scummtr -oh output where control bytes are
			// represented as \NNN decimal escapes. Decode these to raw bytes so
			// the mapping keys match speech.info's raw-byte EN slots.
			// After decoding, \255\003 becomes the two raw bytes 0xFF 0x03.
			enDecoded := string(DecodeScummtrEscapes(enText))
			pageBreak := "\xff\x03"
			enParts := strings.Split(enDecoded, pageBreak)
			// SV text in swedish.txt uses the same \255\003 literal notation.
			svParts := strings.Split(svText, `\255\003`)
			for i, enPart := range enParts {
				if i >= len(svParts) {
					break
				}
				svPart := svParts[i]
				if strings.TrimSpace(enPart) != "" && strings.TrimSpace(svPart) != "" {
					mapping[enPart] = appendDistinct(mapping[enPart], ScummBytes(svPart))
				}
			}
		}
	}
	return mapping
}

// setupScummtr extracts the scummtr binary and returns its path, temp dir path,
// a cleanup function, and any error.
func setupScummtr() (scummtrPath, tmpDir string, cleanup func(), err error) {
	var scummtrBin []byte
	var scummtrName string
	switch runtime.GOOS {
	case "linux":
		scummtrBin = scummtrLinux
		scummtrName = "scummtr"
	case "darwin":
		scummtrBin = scummtrDarwin
		scummtrName = "scummtr"
	case "windows":
		scummtrBin = scummtrWindows
		scummtrName = "scummtr.exe"
	default:
		return "", "", func() {}, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	tmpDir, err = os.MkdirTemp("", "scummtr-*")
	if err != nil {
		return "", "", func() {}, err
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	scummtrPath = filepath.Join(tmpDir, scummtrName)
	if err = os.WriteFile(scummtrPath, scummtrBin, 0755); err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	return scummtrPath, tmpDir, cleanup, nil
}

// runScummtrExtract extracts all game strings with headers to outputPath.
func runScummtrExtract(scummtrPath, gameDir, outputPath string) error {
	cmd := exec.Command(scummtrPath, "-g", "monkeycdalt", "-p", gameDir, "-oh", "-f", outputPath)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scummtr extract: %w\n%s", err, errBuf.String())
	}
	return nil
}

// InjectTranslation injects a translation file into the classic SCUMM game files
// in gameDir. The directory must contain MONKEY1.000 and MONKEY1.001 named exactly
// in uppercase, as required by scummtr.
//
// translationPath must be a scummtr header-format file ("[room:TYPE#resnum]text"),
// covering all entries for any resource it touches. Swedish UTF-8 characters are
// pre-converted to SCUMM escape codes before injection.
//
// On success the files in gameDir are modified in place.
func InjectTranslation(gameDir, translationPath string) error {
	scummtrPath, tmpDir, cleanup, err := setupScummtr()
	if err != nil {
		return err
	}
	defer cleanup()

	encoded, err := encodeForScummtr(translationPath)
	if err != nil {
		return fmt.Errorf("encoding translation: %w", err)
	}

	encodedPath := filepath.Join(tmpDir, "monkey1_encoded.txt")
	if err := os.WriteFile(encodedPath, encoded, 0644); err != nil {
		return err
	}

	// Run scummtr injection.
	//
	// Flags:
	//   -g monkeycdalt  game ID for the MONKEY1.000 file variant
	//   -p              path to directory containing MONKEY1.000 + MONKEY1.001
	//   -i              inject mode: import text INTO the game files
	//   -h              strip [room:TYPE#resnum] header prefixes from each line
	//   -f              path to the pre-encoded translation file
	//
	// Note: -r (raw mode) is required — without it scummtr does in-place replacement
	// which silently overwrites adjacent script bytecode when a Swedish string is
	// longer than its English original. 1843 of our 4437 strings overflow in
	// non-raw mode, corrupting MONKEY1.001 script data and causing the SE engine
	// to crash. Raw mode re-encodes the full LFLF block and adjusts all offsets
	// correctly when string lengths change. (An earlier note claimed raw mode fails
	// on opcode 0x29 in "MI1SE scripts" — that failure is specific to the SE's own
	// enhanced scripts in the PAK, not the embedded classic MONKEY1.001.)
	// Note: -c (Windows-1252 mode) is intentionally omitted — Swedish characters
	// have already been converted to SCUMM escape codes by encodeBytes.
	// Note: -w (CRLF) is intentionally omitted — the file uses Unix LF line endings.
	// Note: -A aov is intentionally omitted — we want verb/object/actor strings
	// injected as well so that menu items (e.g. "Öppna") are translated.
	cmd := exec.Command(
		scummtrPath,
		"-g", "monkeycdalt",
		"-p", gameDir,
		"-ihr",
		"-f", encodedPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
