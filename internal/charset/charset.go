// Package charset patches Swedish glyph data into CHAR_0001 and CHAR_0003
// inside the MONKEY1.000/001 classic game files using the scummrp tool.
//
// scummrp handles all SCUMM block-level housekeeping: XOR encoding, LFLF/LECF
// size fields, the LOFF room-offset table, and the DCHR charset directory.
package charset

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed assets/char_0001_patched.bin
var patchedChar0001 []byte

//go:embed assets/char_0003_patched.bin
var patchedChar0003 []byte

//go:embed assets/scummrp-linux-x64
var scummrpLinux []byte

//go:embed assets/scummrp-darwin-x64
var scummrpDarwin []byte

//go:embed assets/scummrp-windows-x64.exe
var scummrpWindows []byte

// Patch replaces CHAR_0001 and CHAR_0003 in the MONKEY1.000/001 files in
// gameDir with embedded Swedish-glyph versions. scummrp handles all
// block-level housekeeping: XOR encoding, LFLF/LECF sizes, LOFF table, DCHR.
//
// gameDir must contain MONKEY1.000 and MONKEY1.001 in uppercase, as required
// by scummrp's monkeycdalt game ID.
func Patch(gameDir string) error {
	var scummrpBin []byte
	var scummrpName string
	switch runtime.GOOS {
	case "linux":
		scummrpBin = scummrpLinux
		scummrpName = "scummrp"
	case "darwin":
		scummrpBin = scummrpDarwin
		scummrpName = "scummrp"
	case "windows":
		scummrpBin = scummrpWindows
		scummrpName = "scummrp.exe"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	tmpDir, err := os.MkdirTemp("", "scummrp-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	scummrpPath := filepath.Join(tmpDir, scummrpName)
	if err := os.WriteFile(scummrpPath, scummrpBin, 0755); err != nil {
		return err
	}

	dumpDir := filepath.Join(tmpDir, "dump")

	run := func(args ...string) error {
		cmd := exec.Command(scummrpPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Export only CHAR blocks to dump directory.
	if err := run("-g", "monkeycdalt", "-p", gameDir, "-t", "CHAR", "-od", dumpDir); err != nil {
		return fmt.Errorf("scummrp export: %w", err)
	}

	// Replace CHAR_0001 and CHAR_0003 with the Swedish-glyph versions.
	charDir := filepath.Join(dumpDir, "DISK_0001", "LECF", "LFLF_0010")
	if err := os.WriteFile(filepath.Join(charDir, "CHAR_0001"), patchedChar0001, 0644); err != nil {
		return fmt.Errorf("write CHAR_0001: %w", err)
	}
	if err := os.WriteFile(filepath.Join(charDir, "CHAR_0003"), patchedChar0003, 0644); err != nil {
		return fmt.Errorf("write CHAR_0003: %w", err)
	}

	// Import patched CHAR blocks back; scummrp updates all size fields and tables.
	if err := run("-g", "monkeycdalt", "-p", gameDir, "-t", "CHAR", "-id", dumpDir); err != nil {
		return fmt.Errorf("scummrp import: %w", err)
	}

	return nil
}
