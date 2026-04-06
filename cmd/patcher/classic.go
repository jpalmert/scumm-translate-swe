package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"scumm-patcher/internal/backup"
)

// runClassicPatch is the testable entry point for the Classic CD-ROM patching pipeline.
//
// Accepts both classic naming conventions (MONKEY1.000/001 and MONKEY.000/001,
// upper or lowercase). Files are copied to a temp directory as MONKEY1.000/001
// (uppercase) before patching, because scummtr and scummrp require that name.
func runClassicPatch(gameDir, translationArg string) error {
	translationPath, err := findTranslationFile(translationArg)
	if err != nil {
		return err
	}

	fmt.Printf("Platform:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Game dir:    %s\n", gameDir)
	fmt.Printf("Translation: %s\n\n", translationPath)

	// --- Find game files (MONKEY1 or MONKEY, upper or lowercase) ---
	path000, err := findGameFile(gameDir,
		"MONKEY1.000", "monkey1.000",
		"MONKEY.000", "monkey.000")
	if err != nil {
		return err
	}
	path001, err := findGameFile(gameDir,
		"MONKEY1.001", "monkey1.001",
		"MONKEY.001", "monkey.001")
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

	// --- Copy to temp dir as MONKEY1.000/001 (scummtr/scummrp require uppercase MONKEY1) ---
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
	if err := os.WriteFile(filepath.Join(tmpDir, "MONKEY1.000"), data000, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "MONKEY1.001"), data001, 0644); err != nil {
		return err
	}

	// --- Patch: inject translation, CHAR blocks, verb layout ---
	if err := patchClassicFiles(tmpDir, translationPath); err != nil {
		return err
	}

	// --- Write patched files back ---
	fmt.Println("\n==> Writing patched files...")
	patched000, err := os.ReadFile(filepath.Join(tmpDir, "MONKEY1.000"))
	if err != nil {
		return err
	}
	patched001, err := os.ReadFile(filepath.Join(tmpDir, "MONKEY1.001"))
	if err != nil {
		return err
	}
	fmt.Printf("    %s: %d bytes (was %d)\n", filepath.Base(path000), len(patched000), len(data000))
	fmt.Printf("    %s: %d bytes (was %d)\n", filepath.Base(path001), len(patched001), len(data001))

	if err := os.WriteFile(path000, patched000, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path000, err)
	}
	if err := os.WriteFile(path001, patched001, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path001, err)
	}

	return nil
}

// findGameFile returns the path of the first existing file from the given names in dir.
func findGameFile(dir string, names ...string) (string, error) {
	for _, name := range names {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not found in %s", names[0], dir)
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
