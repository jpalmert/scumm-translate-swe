// MI1 Classic Swedish Translation Patcher
//
// Patches MONKEY1.000 and MONKEY1.001 in-place with the Swedish translation.
// Creates MONKEY1.000.bak and MONKEY1.001.bak before modifying the originals.
//
// The translation file (monkey1_swe.txt) is loaded from the same directory as
// this executable by default. You can also pass its path explicitly.
//
// Usage:
//
//	classic-patcher <game_dir> [translation_file]
//
//	game_dir          Directory containing MONKEY1.000 and MONKEY1.001.
//	                  On Linux, lowercase monkey1.000/001 are also accepted.
//	translation_file  Path to monkey1_swe.txt (default: next to this executable).
//
// After patching, play via ScummVM and set the game language to French.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/classic"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	gameDir := os.Args[1]
	translationArg := ""
	if len(os.Args) >= 3 {
		translationArg = os.Args[2]
	}

	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Game dir: %s\n\n", gameDir)

	if err := runClassicPatch(gameDir, translationArg); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nDone!")
	fmt.Println("Play in ScummVM and set the game language to French to see Swedish text.")
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "MI1 Classic Swedish Translation Patcher\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s <game_dir> [translation_file]\n\n", exe)
	fmt.Fprintf(os.Stderr, "  game_dir          Directory containing MONKEY1.000 and MONKEY1.001\n")
	fmt.Fprintf(os.Stderr, "  translation_file  Path to monkey1_swe.txt\n")
	fmt.Fprintf(os.Stderr, "                    (default: monkey1_swe.txt next to this executable)\n\n")
	fmt.Fprintf(os.Stderr, "Creates MONKEY1.000.bak and MONKEY1.001.bak before patching.\n")
	fmt.Fprintf(os.Stderr, "After patching, set the ScummVM game language to French.\n")
}

// runClassicPatch is the testable entry point for the patching pipeline.
func runClassicPatch(gameDir, translationArg string) error {
	// --- Resolve translation file ---
	translationPath, err := findTranslationFile(translationArg)
	if err != nil {
		return err
	}
	fmt.Printf("Translation: %s\n\n", translationPath)

	// --- Find game files (accept both upper and lower case) ---
	path000, err := findGameFile(gameDir, "MONKEY1.000", "monkey1.000")
	if err != nil {
		return err
	}
	path001, err := findGameFile(gameDir, "MONKEY1.001", "monkey1.001")
	if err != nil {
		return err
	}
	fmt.Printf("Found: %s (%d bytes)\n", path000, fileSize(path000))
	fmt.Printf("Found: %s (%d bytes)\n\n", path001, fileSize(path001))

	// --- Backup originals ---
	fmt.Println("==> Creating backups...")
	bak000, err := backup.Create(path000)
	if err != nil {
		return fmt.Errorf("backup %s: %w", path000, err)
	}
	bak001, err := backup.Create(path001)
	if err != nil {
		return fmt.Errorf("backup %s: %w", path001, err)
	}
	fmt.Printf("    %s\n", bak000)
	fmt.Printf("    %s\n", bak001)

	// --- Copy to temp dir as uppercase (scummtr requires uppercase filenames) ---
	tmpDir, err := os.MkdirTemp("", "mi1-classic-patcher-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	data000, err := os.ReadFile(path000)
	if err != nil {
		return err
	}
	data001, err := os.ReadFile(path001)
	if err != nil {
		return err
	}
	tmp000 := filepath.Join(tmpDir, "MONKEY1.000")
	tmp001 := filepath.Join(tmpDir, "MONKEY1.001")
	if err := os.WriteFile(tmp000, data000, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(tmp001, data001, 0644); err != nil {
		return err
	}

	// --- Inject translation ---
	fmt.Println("\n==> Injecting Swedish translation...")
	if err := classic.InjectTranslation(tmpDir, translationPath); err != nil {
		return fmt.Errorf("translation injection failed: %w", err)
	}

	// --- Write patched files back to original paths ---
	fmt.Println("\n==> Writing patched files...")
	patched000, err := os.ReadFile(tmp000)
	if err != nil {
		return err
	}
	patched001, err := os.ReadFile(tmp001)
	if err != nil {
		return err
	}
	fmt.Printf("    MONKEY1.000: %d bytes (was %d)\n", len(patched000), len(data000))
	fmt.Printf("    MONKEY1.001: %d bytes (was %d)\n", len(patched001), len(data001))

	if err := os.WriteFile(path000, patched000, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path000, err)
	}
	if err := os.WriteFile(path001, patched001, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path001, err)
	}

	return nil
}

// findTranslationFile resolves the translation file path. If explicit is non-empty
// it is used directly. Otherwise monkey1_swe.txt is looked up next to the executable.
func findTranslationFile(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("translation file not found: %s", explicit)
		}
		return explicit, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}
	p := filepath.Join(filepath.Dir(exe), "monkey1_swe.txt")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", fmt.Errorf(
		"translation file not found\n"+
			"  Expected: %s\n"+
			"  Or pass the path explicitly: %s <game_dir> <translation_file>",
		p, filepath.Base(os.Args[0]))
}

// findGameFile looks for name1 then name2 in dir, returning the path of whichever exists.
func findGameFile(dir, name1, name2 string) (string, error) {
	for _, name := range []string{name1, name2} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not found in %s", name1, dir)
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
