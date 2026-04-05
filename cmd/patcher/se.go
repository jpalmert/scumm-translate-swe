package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/charset"
	"scumm-patcher/internal/classic"
	"scumm-patcher/internal/font"
	"scumm-patcher/internal/pak"
)

// runSEPatch is the testable entry point for the SE patching pipeline.
//
// outputPAK: if empty, patches inputPAK in-place (with backup).
// translationArg: if empty, monkey1.txt is looked up next to the executable.
func runSEPatch(inputPAK, outputPAK, translationArg string) error {
	translationPath, err := findTranslationFile(translationArg)
	if err != nil {
		return err
	}

	fmt.Printf("Platform:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Input:       %s\n", inputPAK)
	if outputPAK == "" {
		fmt.Printf("Output:      %s (in-place with backup)\n", inputPAK)
	} else {
		fmt.Printf("Output:      %s\n", outputPAK)
	}
	fmt.Printf("Translation: %s\n\n", translationPath)

	inPlace := outputPAK == ""
	if inPlace {
		outputPAK = inputPAK
	}

	// --- Step 1: Read PAK ---
	fmt.Println("==> Reading PAK...")
	hdr, indexBlob, namesBlob, entries, err := pak.Read(inputPAK)
	if err != nil {
		return fmt.Errorf("reading PAK: %w", err)
	}
	fmt.Printf("    %d files\n", len(entries))

	var entry000, entry001 *pak.Entry
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			entry000 = e
		case "classic/en/monkey1.001":
			entry001 = e
		}
	}
	if entry000 == nil {
		return fmt.Errorf("classic/en/monkey1.000 not found — is this really Monkey1.pak?")
	}
	if entry001 == nil {
		return fmt.Errorf("classic/en/monkey1.001 not found — is this really Monkey1.pak?")
	}
	fmt.Printf("    MONKEY1.000: %d bytes\n", len(entry000.Data))
	fmt.Printf("    MONKEY1.001: %d bytes\n", len(entry001.Data))

	// --- Step 2: Backup if patching in-place ---
	if inPlace {
		fmt.Println("\n==> Creating backup...")
		bakPath, err := backup.Create(inputPAK)
		if err != nil {
			return fmt.Errorf("backup: %w", err)
		}
		fmt.Printf("    %s\n", bakPath)
	}

	// --- Step 3: Extract classic files to temp dir ---
	tmpDir, err := os.MkdirTemp("", "mi1-patcher-se-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	path000 := filepath.Join(tmpDir, "MONKEY1.000")
	path001 := filepath.Join(tmpDir, "MONKEY1.001")
	if err := os.WriteFile(path000, entry000.Data, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(path001, entry001.Data, 0644); err != nil {
		return err
	}

	// --- Step 4: Inject translation ---
	fmt.Println("\n==> Injecting Swedish translation...")
	if err := classic.InjectTranslation(tmpDir, translationPath); err != nil {
		return fmt.Errorf("translation injection failed: %w", err)
	}

	// --- Step 5: Read patched files back ---
	patched000, err := os.ReadFile(path000)
	if err != nil {
		return err
	}
	patched001, err := os.ReadFile(path001)
	if err != nil {
		return err
	}
	fmt.Printf("    MONKEY1.000: %d bytes (was %d)\n", len(patched000), len(entry000.Data))
	fmt.Printf("    MONKEY1.001: %d bytes (was %d)\n", len(patched001), len(entry001.Data))

	// --- Step 5a: Update MONKEY1.000 charset offset table ---
	fmt.Println("\n==> Updating charset offset table in MONKEY1.000...")
	patched000, err = charset.PatchMonkey1000(patched000)
	if err != nil {
		return fmt.Errorf("MONKEY1.000 charset offset update failed: %w", err)
	}
	fmt.Printf("    MONKEY1.000: %d bytes (after offset update)\n", len(patched000))

	entry000.Data = patched000
	entry001.Data = patched001

	// --- Step 5b: Patch classic charset (add Swedish glyphs to CHAR_0001/0003) ---
	fmt.Println("\n==> Patching classic charset (adding Swedish glyphs)...")
	charsetPatched, err := charset.PatchMonkey1001(entry001.Data)
	if err != nil {
		return fmt.Errorf("charset patching failed: %w", err)
	}
	entry001.Data = charsetPatched
	fmt.Printf("    MONKEY1.001: %d bytes (after charset patch)\n", len(entry001.Data))

	// --- Step 6: Patch font lookup tables ---
	fmt.Println("\n==> Patching font lookup tables...")
	fontCount, err := remapFontEntries(entries)
	if err != nil {
		return fmt.Errorf("font patching failed: %w", err)
	}
	fmt.Printf("    Patched %d font files\n", fontCount)

	// --- Step 7: Repack PAK ---
	fmt.Println("\n==> Repacking PAK...")
	if err := pak.Write(outputPAK, hdr, indexBlob, namesBlob, entries); err != nil {
		return fmt.Errorf("writing PAK: %w", err)
	}
	fmt.Printf("    Written: %s\n", outputPAK)

	return nil
}

// remapFontEntries patches the glyph lookup table in every .font entry.
// Returns the number of font files patched.
func remapFontEntries(entries []*pak.Entry) (int, error) {
	count := 0
	for _, e := range entries {
		if !strings.HasSuffix(strings.ToLower(e.Name), ".font") {
			continue
		}
		patched, err := font.RemapLookup(e.Data, font.SwedishRemapping)
		if err != nil {
			return count, fmt.Errorf("%s: %w", e.Name, err)
		}
		e.Data = patched
		count++
	}
	return count, nil
}
