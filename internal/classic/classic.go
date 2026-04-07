// Package classic provides scummtr-based text injection for classic SCUMM game files.
//
// The embedded scummtr binary is extracted to a temporary directory at run time
// and invoked with the appropriate flags for Monkey Island 1.
package classic

import (
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
var scummCharMap = []struct {
	from string
	to   string
}{
	{"Å", `\197`},
	{"Ä", `\196`},
	{"Ö", `\214`},
	{"å", `\229`},
	{"ä", `\228`},
	{"ö", `\246`},
	{"é", `\130`},
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
	s := string(data)
	for _, m := range scummCharMap {
		s = strings.ReplaceAll(s, m.from, m.to)
	}
	trailingNL := strings.HasSuffix(s, "\n")
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	filtered := lines[:0]
	for _, line := range lines {
		if i := strings.IndexByte(line, ']'); !(i >= 0 && strings.HasPrefix(line, "[") && strings.TrimSpace(line[i+1:]) == "") {
			filtered = append(filtered, line)
		}
	}
	result := strings.Join(filtered, "\n")
	if trailingNL {
		result += "\n"
	}
	return []byte(result), nil
}

// InjectTranslation injects a translation file into the classic SCUMM game files
// in gameDir. The directory must contain MONKEY1.000 and MONKEY1.001 named exactly
// in uppercase, as required by scummtr.
//
// translationPath must be a Windows-1252 encoded scummtr-format text file.
// Swedish characters (åäöÅÄÖé) are pre-converted to SCUMM escape codes before
// injection so that the classic engine renders them correctly.
//
// On success the files in gameDir are modified in place.
func InjectTranslation(gameDir, translationPath string) error {
	// Select the scummtr binary for the current platform.
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
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Extract scummtr binary to a temp directory.
	tmpDir, err := os.MkdirTemp("", "scummtr-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	scummtrPath := filepath.Join(tmpDir, scummtrName)
	if err := os.WriteFile(scummtrPath, scummtrBin, 0755); err != nil {
		return err
	}

	// Pre-encode Swedish characters as SCUMM escape codes.
	// scummtr's -c flag does not reliably map Windows-1252 Swedish chars to the
	// correct SCUMM internal codes for monkeycdalt, so we do it ourselves first.
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
	// handles variable-length strings correctly with encodeForScummtr pre-encoding.
	// Note: -c (Windows-1252 mode) is intentionally omitted — Swedish characters
	// have already been converted to SCUMM escape codes by encodeForScummtr.
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
