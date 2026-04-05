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

// InjectTranslation injects a translation file into the classic SCUMM game files
// in gameDir. The directory must contain MONKEY1.000 and MONKEY1.001 named exactly
// in uppercase, as required by scummtr.
//
// translationPath must be a scummtr-format text file encoded in Windows-1252 with
// Unix LF line endings (do NOT use -w / CRLF when generating this file).
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

	// Run scummtr injection.
	//
	// Flags:
	//   -g monkeycdalt  game ID for the MONKEY1.000 file variant
	//   -p              path to directory containing MONKEY1.000 + MONKEY1.001
	//   -c              Windows-1252 character encoding (Swedish chars: å ä ö)
	//   -A aov          protect actor/object/verb names from accidental overwrite
	//   -i              inject mode: import text INTO the game files
	//   -f              path to the translation file
	//
	// Note: -w (CRLF) is intentionally omitted. The translation file uses Unix LF
	// endings. The -w flag makes scummtr expect CRLF when reading, causing misparse.
	cmd := exec.Command(
		scummtrPath,
		"-g", "monkeycdalt",
		"-p", gameDir,
		"-c",
		"-A", "aov",
		"-i",
		"-f", translationPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
