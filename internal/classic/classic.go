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
}

// ScummBytes converts UTF-8 Swedish text to its SCUMM byte representation.
// Used for building speech.info EN slot content that matches injected game strings.
func ScummBytes(text string) []byte {
	var b []byte
	for _, r := range text {
		if mapped, ok := scummByteMap[r]; ok {
			b = append(b, mapped)
		} else if r < 0x80 {
			b = append(b, byte(r))
		}
		// unknown non-ASCII chars dropped; shouldn't appear in MI1 translations
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

// BuildSpeechMapping extracts the original English text from the classic game
// files in gameDir, then parses the Swedish translation at translationPath, and
// returns a map from each original English string to its SCUMM-encoded Swedish
// byte representation.
//
// This map is used to update speech.info EN slots so that the SE engine can
// match voiced lines to audio cues after the Swedish text has been injected.
func BuildSpeechMapping(gameDir, translationPath string) (map[string][]byte, error) {
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

// swordFightResources lists the resource headers whose strings are sword-fight
// insults and comebacks. These are intentionally excluded from the speech
// mapping because the Swedish translation uses non-literal creative rewrites;
// matching the old English audio to the new Swedish text would be misleading.
var swordFightResources = map[string]bool{
	"[088:SCRP#0085]": true, // insults
	"[088:SCRP#0086]": true, // comebacks
}

// buildSpeechMapping builds the EN→SCUMM_bytes mapping from raw data slices.
// Both files use scummtr header format: "[room:TYPE#resnum]text".
// Entries are matched positionally within each resource.
// Sword-fight insult/comeback resources are excluded (see swordFightResources).
func buildSpeechMapping(enData, svData []byte) map[string][]byte {
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
	mapping := make(map[string][]byte)
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
			// Split both EN and SV on \255\003 (page-break escape) so each sentence
			// gets its own mapping entry. Single-page strings split into one part.
			enParts := strings.Split(enText, `\255\003`)
			svParts := strings.Split(svText, `\255\003`)
			for i, enPart := range enParts {
				if i >= len(svParts) {
					break
				}
				svPart := svParts[i]
				if strings.TrimSpace(enPart) != "" && strings.TrimSpace(svPart) != "" {
					mapping[enPart] = ScummBytes(svPart)
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
	// Note: -r (raw mode) is intentionally omitted — raw mode triggers a full
	// script re-encode when strings change length, which fails on opcode 0x29
	// present in some MI1SE scripts. Non-raw mode uses in-place replacement and
	// handles variable-length strings correctly with encodeBytes pre-encoding.
	// Note: -c (Windows-1252 mode) is intentionally omitted — Swedish characters
	// have already been converted to SCUMM escape codes by encodeBytes.
	// Note: -w (CRLF) is intentionally omitted — the file uses Unix LF line endings.
	// Note: -A aov is intentionally omitted — we want verb/object/actor strings
	// injected as well so that menu items (e.g. "Öppna") are translated.
	cmd := exec.Command(
		scummtrPath,
		"-g", "monkeycdalt",
		"-p", gameDir,
		"-ih",
		"-f", encodedPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
