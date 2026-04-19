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
	{"™", `\153`},
	{"\u00a0", `\250`},
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
//
// A valid opcode prefix is exactly "(XX)" where XX is two hex digits or
// underscores, e.g. "(D8)", "(__)", "(14)". The closing ")" must be at
// position 3 — anything else is literal game text that starts with "(".
func stripOpcode(text string) string {
	if len(text) >= 4 && text[0] == '(' && text[3] == ')' {
		return text[4:]
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
