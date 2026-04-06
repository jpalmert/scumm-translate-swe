package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/charset"
	"scumm-patcher/internal/classic"
)

// runClassicPatch is the testable entry point for the Classic CD-ROM patching pipeline.
//
// Classic mode operates directly on the MONKEY1.000 and MONKEY1.001 files used by
// ScummVM. The pipeline has three stages:
//  1. Inject Swedish text strings (internal/classic — scummtr, game ID monkeycdalt).
//  2. Patch CHAR blocks with Swedish glyph bitmaps (internal/charset — scummrp).
//     The five CHAR blocks cover all on-screen fonts; without this step Swedish
//     character codes render as ASCII punctuation or nothing.
//  3. Write the patched files back to gameDir, replacing the originals.
//
// The game files are copied to a temp directory with uppercase names before
// processing because scummtr and scummrp require MONKEY1.000/001 in uppercase.
func runClassicPatch(gameDir, translationArg string) error {
	translationPath, err := findTranslationFile(translationArg)
	if err != nil {
		return err
	}

	fmt.Printf("Platform:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Game dir:    %s\n", gameDir)
	fmt.Printf("Translation: %s\n\n", translationPath)

	// --- Find game files (accept upper or lowercase) ---
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
	if errors.Is(err, backup.ErrBackupExists) {
		fmt.Printf("    WARNING: %s already exists from a previous run — using it as-is.\n", bak000)
	} else if err != nil {
		return fmt.Errorf("backup %s: %w", path000, err)
	} else {
		fmt.Printf("    %s\n", bak000)
	}
	bak001, err := backup.Create(path001)
	if errors.Is(err, backup.ErrBackupExists) {
		fmt.Printf("    WARNING: %s already exists from a previous run — using it as-is.\n", bak001)
	} else if err != nil {
		return fmt.Errorf("backup %s: %w", path001, err)
	} else {
		fmt.Printf("    %s\n", bak001)
	}

	// --- Copy to temp dir as uppercase (scummtr requires uppercase filenames) ---
	tmpDir, err := os.MkdirTemp("", "mi1-patcher-classic-*")
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

	// --- Patch CHAR blocks (Swedish glyph data) ---
	fmt.Println("\n==> Patching CHAR blocks (Swedish glyph data)...")
	if err := charset.Patch(tmpDir); err != nil {
		return fmt.Errorf("charset patch: %w", err)
	}

	// --- Patch verb button layout ---
	fmt.Println("\n==> Patching verb button layout...")
	if err := charset.PatchVerbLayout(tmpDir); err != nil {
		return fmt.Errorf("verb layout patch: %w", err)
	}

	// --- Write patched files back ---
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

// findGameFile looks for name1 then name2 in dir, returning whichever exists.
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
