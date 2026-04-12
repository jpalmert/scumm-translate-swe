package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/font"
	"scumm-patcher/internal/pak"
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

	// --- Step 4: Patch classic files (translation, CHAR blocks, verb layout) ---
	// CHAR blocks are needed for the SE's classic rendering mode (F1 toggle).
	// Verb layout reordering ensures Swedish button labels fit correctly in both
	// classic and SE rendering modes.
	if err := patchClassicFiles(tmpDir, translationPath); err != nil {
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

	// --- Step 8: Disable autosave via tweaks.txt ---
	// The PAK ships with a tweaks.txt containing "SCUMM.Save game,1" — a developer
	// debug setting that triggers an autosave at 5 minutes of game time. With a
	// modified MONKEY1.001 (translation injected, file structure changed by raw-mode
	// re-encoding) the SE engine crashes during that autosave. Zeroing the save
	// value disables the autosave and prevents the crash.
	fmt.Println("\n==> Disabling autosave (tweaks.txt)...")
	disableAutosave(entries)

	// --- Step 9: Repack PAK ---
	fmt.Println("\n==> Repacking PAK...")
	if err := pak.Write(outputPAK, hdr, indexBlob, namesBlob, entries); err != nil {
		return fmt.Errorf("writing PAK: %w", err)
	}
	fmt.Printf("    Written: %s\n", outputPAK)

	return nil
}

// disableAutosave patches the "SCUMM.Save game" line in the tweaks.txt PAK entry
// to value 0, preventing the SE engine's 5-minute autosave from firing.
func disableAutosave(entries []*pak.Entry) {
	for _, e := range entries {
		if strings.ToLower(e.Name) != "tweaks.txt" {
			continue
		}
		lines := strings.Split(string(e.Data), "\n")
		patched := false
		for i, line := range lines {
			if strings.HasPrefix(line, "SCUMM.Save game,") {
				lines[i] = "SCUMM.Save game,0"
				patched = true
			}
		}
		e.Data = []byte(strings.Join(lines, "\n"))
		if patched {
			fmt.Println("    SCUMM.Save game set to 0")
		} else {
			fmt.Println("    tweaks.txt found but no SCUMM.Save game line — skipping")
		}
		return
	}
	fmt.Println("    tweaks.txt not found — skipping")
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
