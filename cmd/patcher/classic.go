package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/classic"
)

// runClassicPatch is the testable entry point for the classic patching pipeline.
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
