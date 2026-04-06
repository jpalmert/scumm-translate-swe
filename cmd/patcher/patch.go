package main

import (
	"fmt"

	"scumm-patcher/internal/charset"
	"scumm-patcher/internal/classic"
)

// patchClassicFiles applies all patches to MONKEY1.000/001 in tmpDir:
//  1. Inject Swedish text strings (scummtr).
//  2. Patch CHAR blocks with Swedish glyph bitmaps (scummrp).
//  3. Reorder verb action buttons for Swedish word lengths (scummrp).
//
// Files in tmpDir must be named MONKEY1.000 and MONKEY1.001 (uppercase).
func patchClassicFiles(tmpDir, translationPath string) error {
	fmt.Println("\n==> Injecting Swedish translation...")
	if err := classic.InjectTranslation(tmpDir, translationPath); err != nil {
		return fmt.Errorf("translation injection failed: %w", err)
	}

	fmt.Println("\n==> Patching CHAR blocks (Swedish glyph data)...")
	if err := charset.Patch(tmpDir); err != nil {
		return fmt.Errorf("charset patch: %w", err)
	}

	fmt.Println("\n==> Patching verb button layout...")
	if err := charset.PatchVerbLayout(tmpDir); err != nil {
		return fmt.Errorf("verb layout patch: %w", err)
	}

	return nil
}
