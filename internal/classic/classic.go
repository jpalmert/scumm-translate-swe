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
// with Swedish characters replaced by their SCUMM escape codes.
// Lines whose [room:TYPE#resnum] header has no content after it are dropped —
// scummtr rejects them with "Empty lines are forbidden".
func encodeForScummtr(translationPath string) ([]byte, error) {
	data, err := os.ReadFile(translationPath)
	if err != nil {
		return nil, err
	}
	return encodeBytes(data), nil
}

// encodeBytes converts UTF-8 Swedish characters in scummtr-format data to
// SCUMM escape codes, and pads empty-content header lines with a single space
// so scummtr does not reject them.
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
			if strings.TrimSpace(line[j+1:]) == "" {
				lines[i] = line[:j+1] + " "
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// mergeTranslation overlays Swedish translations onto the full English
// extraction. For each game string, uses the Swedish text if a translation
// exists for that position within that resource, otherwise keeps the English.
//
// Both files use scummtr header format: "[room:TYPE#resnum]text".
// The SV file may have blank lines (untranslated positions) which are skipped.
// SV lines with a leading (NN) token (legacy monkeycd_swe format) have it stripped.
func mergeTranslation(enData, svData []byte) []byte {
	// Build SV groups: resource_header -> []text in order.
	svGroups := make(map[string][]string)
	for _, line := range strings.Split(string(svData), "\n") {
		j := strings.IndexByte(line, ']')
		if j < 0 || !strings.HasPrefix(line, "[") {
			continue // blank or non-header lines are untranslated entries
		}
		header := line[:j+1]
		text := line[j+1:]
		svGroups[header] = append(svGroups[header], text)
	}

	// Merge: walk EN entries; substitute SV where available.
	svPos := make(map[string]int) // tracks next unused SV entry per resource
	var out []string
	for _, line := range strings.Split(string(enData), "\n") {
		j := strings.IndexByte(line, ']')
		if j < 0 || !strings.HasPrefix(line, "[") {
			out = append(out, line)
			continue
		}
		header := line[:j+1]
		enText := line[j+1:]
		p := svPos[header]
		svPos[header]++

		if svTexts, ok := svGroups[header]; ok && p < len(svTexts) {
			svText := stripParenPrefix(svTexts[p])
			if strings.TrimSpace(svText) != "" {
				out = append(out, header+svText)
				continue
			}
		}
		out = append(out, header+enText)
	}
	return []byte(strings.Join(out, "\n"))
}

// stripParenPrefix removes a leading (N) token if present (e.g. "(27)text" → "text").
// This legacy prefix appeared in monkeycd_swe translations where an older scummtr
// version included the text-color opcode value in the extraction output.
func stripParenPrefix(text string) string {
	if len(text) == 0 || text[0] != '(' {
		return text
	}
	j := strings.IndexByte(text, ')')
	if j < 0 {
		return text
	}
	for _, c := range text[1:j] {
		if c < '0' || c > '9' {
			return text
		}
	}
	return text[j+1:]
}

// BuildSpeechMapping extracts the original English text from the classic game
// files in gameDir, overlays the Swedish translation from translationPath, and
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

// buildSpeechMapping builds the EN→SCUMM_bytes mapping from raw data slices.
func buildSpeechMapping(enData, svData []byte) map[string][]byte {
	svGroups := make(map[string][]string)
	for _, line := range strings.Split(string(svData), "\n") {
		j := strings.IndexByte(line, ']')
		if j < 0 || !strings.HasPrefix(line, "[") {
			continue
		}
		header := line[:j+1]
		svGroups[header] = append(svGroups[header], line[j+1:])
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

		if svTexts, ok := svGroups[header]; ok && p < len(svTexts) {
			svText := stripParenPrefix(svTexts[p])
			if strings.TrimSpace(svText) != "" && strings.TrimSpace(enText) != "" {
				mapping[enText] = ScummBytes(svText)
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
// The injector first extracts the full English text, then overlays the provided
// translation (which may be partial), producing a complete merged translation.
// Swedish UTF-8 characters are pre-encoded to SCUMM escape codes before injection.
//
// On success the files in gameDir are modified in place.
func InjectTranslation(gameDir, translationPath string) error {
	scummtrPath, tmpDir, cleanup, err := setupScummtr()
	if err != nil {
		return err
	}
	defer cleanup()

	// Extract original English text with headers.
	enPath := filepath.Join(tmpDir, "monkey1_en.txt")
	if err := runScummtrExtract(scummtrPath, gameDir, enPath); err != nil {
		return fmt.Errorf("extracting source text: %w", err)
	}

	enData, err := os.ReadFile(enPath)
	if err != nil {
		return err
	}
	svData, err := os.ReadFile(translationPath)
	if err != nil {
		return fmt.Errorf("reading translation: %w", err)
	}

	// Merge Swedish translations onto the English base.
	merged := mergeTranslation(enData, svData)

	// Encode Swedish characters as SCUMM escape codes.
	encoded := encodeBytes(merged)

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
	// handles variable-length strings correctly with encodeForScummtr pre-encoding.
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
