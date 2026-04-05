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
)

// scummCharMap maps Windows-1252 Swedish characters to their SCUMM internal
// escape codes for Monkey Island 1. scummtr does not reliably auto-convert
// these via -c for the monkeycdalt game ID, so we pre-encode them in the
// translation file before injection (matching the monkeycd_swe project approach).
var scummCharMap = []struct {
	from byte
	to   string
}{
	{0xC5, `\091`}, // Å
	{0xC4, `\092`}, // Ä
	{0xD6, `\093`}, // Ö
	{0xE5, `\123`}, // å
	{0xE4, `\124`}, // ä
	{0xF6, `\125`}, // ö
	{0xE9, `\130`}, // é
}

// encodeForScummtr reads a Windows-1252 encoded translation file and returns
// a copy with Swedish characters replaced by their SCUMM escape codes.
func encodeForScummtr(translationPath string) ([]byte, error) {
	data, err := os.ReadFile(translationPath)
	if err != nil {
		return nil, err
	}
	// Replace each special byte with its escape sequence.
	// Work byte-by-byte to avoid multi-byte string issues with Windows-1252.
	out := make([]byte, 0, len(data))
	for _, b := range data {
		replaced := false
		for _, m := range scummCharMap {
			if b == m.from {
				out = append(out, []byte(m.to)...)
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, b)
		}
	}
	return out, nil
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
	//   -A aov          protect actor/object/verb names from accidental overwrite
	//   -i              inject mode: import text INTO the game files
	//   -f              path to the pre-encoded translation file
	//
	// Note: -c (Windows-1252 mode) is intentionally omitted — Swedish characters
	// have already been converted to SCUMM escape codes by encodeForScummtr.
	// Note: -w (CRLF) is intentionally omitted — the file uses Unix LF line endings.
	cmd := exec.Command(
		scummtrPath,
		"-g", "monkeycdalt",
		"-p", gameDir,
		"-A", "aov",
		"-i",
		"-f", encodedPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
