package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/classic"
	"scumm-patcher/internal/font"
	"scumm-patcher/internal/pak"
	"scumm-patcher/internal/speech"
)

// runSEPatch is the testable entry point for the Special Edition patching pipeline.
//
// The SE (Monkey1.pak) embeds the classic MONKEY1.000/001 files under classic/en/.
// Patching requires modifying both the classic files and the SE font tables:
//
//  1. Read the PAK and locate the embedded classic/en/monkey1.000 and .001.
//  2. Extract them to a temp directory as MONKEY1.000/001 (uppercase).
//  3. Inject translation + patch CHAR blocks (verb layout skipped — SE has its own verb UI).
//  4. Patch the glyph lookup table in every .font entry in the PAK.
//  5. Repack the PAK with the modified classic files and updated .font entries.
//
// outputPAK: if empty, patches inputPAK in-place (with backup).
// translationArg: if empty, swedish.txt is looked up next to the executable.
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
		if errors.Is(err, backup.ErrBackupExists) {
			fmt.Printf("    WARNING: %s already exists from a previous run — using it as-is.\n", bakPath)
			// Re-patch: the PAK we just read is already patched from a previous run.
			// Re-read from the backup to get the unmodified originals.
			fmt.Printf("    Re-patch detected: reading originals from backup.\n")
			hdr, indexBlob, namesBlob, entries, err = pak.Read(bakPath)
			if err != nil {
				return fmt.Errorf("reading backup PAK: %w", err)
			}
			// Re-locate classic entries in the freshly read backup.
			entry000, entry001 = nil, nil
			for _, e := range entries {
				switch strings.ToLower(e.Name) {
				case "classic/en/monkey1.000":
					entry000 = e
				case "classic/en/monkey1.001":
					entry001 = e
				}
			}
			if entry000 == nil || entry001 == nil {
				return fmt.Errorf("classic files not found in backup PAK — backup may be corrupt")
			}
			fmt.Printf("    MONKEY1.000: %d bytes (from backup)\n", len(entry000.Data))
			fmt.Printf("    MONKEY1.001: %d bytes (from backup)\n", len(entry001.Data))
		} else if err != nil {
			return fmt.Errorf("backup: %w", err)
		} else {
			fmt.Printf("    %s\n", bakPath)
		}
	}

	// --- Step 3: Extract classic files to temp dir ---
	tmpDir, err := os.MkdirTemp("", "mi1-patcher-se-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "MONKEY1.000"), entry000.Data, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "MONKEY1.001"), entry001.Data, 0644); err != nil {
		return err
	}

	// --- Step 4a: Build speech mapping from ORIGINAL English content ---
	// This must happen before translation injection; after injection the English
	// strings are gone and the mapping cannot be built.
	fmt.Println("\n==> Building speech.info mapping...")
	speechMapping, err := classic.BuildSpeechMapping(tmpDir, translationPath)
	if err != nil {
		return fmt.Errorf("building speech mapping: %w", err)
	}
	fmt.Printf("    %d EN→SV pairs\n", len(speechMapping))

	// --- Step 4: Apply SE-specific classic patches ---
	// Only inject the Swedish translation. CHAR block patching and verb layout
	// patching are skipped for SE because:
	//   - The SE new-graphics mode uses .font files (patched in Step 7), not CHAR blocks.
	//   - Growing CHAR blocks changes MONKEY1.001's internal structure in ways
	//     that can cause the SE engine to crash.
	//   - The SE handles verb display independently; classic verb layout coordinates
	//     are irrelevant for the SE renderer.
	if err := patchSEClassicFiles(tmpDir, translationPath); err != nil {
		return err
	}

	// --- Step 6: Read patched files back into PAK entries ---
	fmt.Println("\n==> Reading patched classic files...")
	patched000, err := os.ReadFile(filepath.Join(tmpDir, "MONKEY1.000"))
	if err != nil {
		return err
	}
	patched001, err := os.ReadFile(filepath.Join(tmpDir, "MONKEY1.001"))
	if err != nil {
		return err
	}
	fmt.Printf("    MONKEY1.000: %d bytes (was %d)\n", len(patched000), len(entry000.Data))
	fmt.Printf("    MONKEY1.001: %d bytes (was %d)\n", len(patched001), len(entry001.Data))

	entry000.Data = patched000
	entry001.Data = patched001

	// --- Step 7: Patch font lookup tables ---
	fmt.Println("\n==> Patching font lookup tables...")
	fontCount, err := remapFontEntries(entries)
	if err != nil {
		return fmt.Errorf("font patching failed: %w", err)
	}
	fmt.Printf("    Patched %d font files\n", fontCount)

	// --- Step 8: Repack PAK ---
	fmt.Println("\n==> Repacking PAK...")
	if err := pak.Write(outputPAK, hdr, indexBlob, namesBlob, entries); err != nil {
		return fmt.Errorf("writing PAK: %w", err)
	}
	fmt.Printf("    Written: %s\n", outputPAK)

	// --- Step 9: Patch speech.info ---
	// speech.info lives next to the PAK in an audio/ subdirectory.
	// The SE engine matches voiced lines by looking up the current string from
	// MONKEY1.001 in speech.info's EN slot. After Swedish injection, the EN slots
	// must also contain Swedish bytes so the lookup succeeds.
	// speech.info is optional — skip silently if not present.
	speechInfoPath := filepath.Join(filepath.Dir(outputPAK), "audio", "speech.info")
	if _, err := os.Stat(speechInfoPath); err == nil {
		fmt.Println("\n==> Patching speech.info...")
		n, err := speech.Patch(speechInfoPath, speechMapping)
		if err != nil {
			return fmt.Errorf("speech.info patch: %w", err)
		}
		fmt.Printf("    Updated %d entries\n", n)
	} else {
		fmt.Println("\n==> speech.info not found — skipping (audio/speech.info not present next to PAK)")
	}

	return nil
}

// patchSEClassicFiles injects the Swedish translation into MONKEY1.000/001 in
// tmpDir. CHAR block patching and verb layout patching are skipped for SE:
// the SE new-graphics mode uses .font files (patched separately), and the SE's
// verb UI does not use the classic SCRP verb layout.
func patchSEClassicFiles(tmpDir, translationPath string) error {
	fmt.Println("\n==> Injecting Swedish translation...")
	if err := classic.InjectTranslation(tmpDir, translationPath); err != nil {
		return fmt.Errorf("translation injection failed: %w", err)
	}
	return nil
}

// runListPAK prints all entry names from a PAK file, one per line.
func runListPAK(pakPath string) error {
	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		return fmt.Errorf("reading PAK: %w", err)
	}
	fmt.Printf("%d entries in %s\n\n", len(entries), pakPath)
	for _, e := range entries {
		fmt.Printf("%8d  %s\n", len(e.Data), e.Name)
	}
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
