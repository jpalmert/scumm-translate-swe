package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/font"
	"scumm-patcher/internal/hints"
	"scumm-patcher/internal/pak"
	"scumm-patcher/internal/uitext"
)

// runSEPatch is the testable entry point for the Special Edition patching pipeline.
//
// The SE (Monkey1.pak) embeds the classic MONKEY1.000/001 files under classic/en/.
// Patching requires modifying both the classic files and the SE font tables:
//
//  1. Read the PAK and locate the embedded classic/en/monkey1.000 and .001.
//  2. Extract them to a temp directory as MONKEY1.000/001 (uppercase).
//  3. Inject translation + patch CHAR blocks + reorder verb layout (needed for F1 classic mode).
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

	entry000, entry001, err := findClassicEntries(entries)
	if err != nil {
		return err
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
			entry000, entry001, err = findClassicEntries(entries)
			if err != nil {
				return fmt.Errorf("backup PAK: %w", err)
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

	// --- Step 5: Read patched files back into PAK entries ---
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

	// --- Step 6: Patch font lookup tables ---
	fmt.Println("\n==> Patching font lookup tables...")
	fontCount, err := remapFontEntries(entries)
	if err != nil {
		return fmt.Errorf("font patching failed: %w", err)
	}
	fmt.Printf("    Patched %d font files\n", fontCount)

	// --- Step 6b: Patch SE hint text ---
	hintsPath := findOptionalSEFile(translationPath, "hints_swedish.txt")
	if hintsPath != "" {
		fmt.Println("\n==> Patching SE hint text...")
		n, err := patchHintEntries(entries, hintsPath)
		if err != nil {
			return fmt.Errorf("hint patching failed: %w", err)
		}
		fmt.Printf("    Replaced %d hint strings\n", n)
	} else {
		fmt.Println("\n    hints_swedish.txt not found — skipping hint patching")
	}

	// --- Step 7: Repack PAK ---
	fmt.Println("\n==> Repacking PAK...")
	if err := pak.Write(outputPAK, hdr, indexBlob, namesBlob, entries); err != nil {
		return fmt.Errorf("writing PAK: %w", err)
	}
	fmt.Printf("    Written: %s\n", outputPAK)

	// --- Step 8: Patch uiText.info ---
	uitextTransPath := findOptionalSEFile(translationPath, "uitext_swedish.txt")
	if uitextTransPath != "" {
		pakDir := filepath.Dir(inputPAK)
		uiTextPath := filepath.Join(pakDir, "localization", "uiText.info")
		if _, err := os.Stat(uiTextPath); err == nil {
			fmt.Println("\n==> Patching SE UI text...")
			if err := patchUIText(uiTextPath, uitextTransPath); err != nil {
				return fmt.Errorf("UI text patching failed: %w", err)
			}
		} else {
			fmt.Println("\n    localization/uiText.info not found — skipping UI text patching")
		}
	} else {
		fmt.Println("\n    uitext_swedish.txt not found — skipping UI text patching")
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

// findClassicEntries locates the classic/en/monkey1.000 and .001 entries in
// a PAK entry list. Returns an error if either is missing.
func findClassicEntries(entries []*pak.Entry) (entry000, entry001 *pak.Entry, err error) {
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			entry000 = e
		case "classic/en/monkey1.001":
			entry001 = e
		}
	}
	if entry000 == nil {
		return nil, nil, fmt.Errorf("classic/en/monkey1.000 not found — is this really Monkey1.pak?")
	}
	if entry001 == nil {
		return nil, nil, fmt.Errorf("classic/en/monkey1.001 not found — is this really Monkey1.pak?")
	}
	return entry000, entry001, nil
}

// patchHintEntries finds the hints/monkey1.hints.csv entry in the PAK,
// replaces English strings with Swedish translations, and updates the
// entry data in-place. Returns the number of strings replaced.
func patchHintEntries(entries []*pak.Entry, hintsTransPath string) (int, error) {
	// Find the hints entry.
	var hintsEntry *pak.Entry
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name), ".hints.csv") {
			hintsEntry = e
			break
		}
	}
	if hintsEntry == nil {
		fmt.Println("    No .hints.csv entry in PAK — nothing to patch")
		return 0, nil
	}

	// Parse the hints binary.
	h, err := hints.Parse(hintsEntry.Data)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", hintsEntry.Name, err)
	}

	// Load translations.
	replacements, err := loadHintsTranslation(hintsTransPath)
	if err != nil {
		return 0, err
	}
	if len(replacements) == 0 {
		return 0, nil
	}

	// Apply replacements.
	if err := h.ReplaceStrings(replacements); err != nil {
		return 0, err
	}

	hintsEntry.Data = h.Serialize()
	return len(replacements), nil
}

// patchUIText reads a uiText.info file, replaces English strings with
// Swedish translations from uitextTransPath, creates a backup, and writes
// the patched file back.
func patchUIText(uiTextPath, uitextTransPath string) error {
	// Create backup.
	bakPath, err := backup.Create(uiTextPath)
	if errors.Is(err, backup.ErrBackupExists) {
		fmt.Printf("    WARNING: %s already exists — re-patching from backup.\n", bakPath)
		// Re-read from backup to get unmodified originals.
		uiTextPath = bakPath
	} else if err != nil {
		return fmt.Errorf("backup uiText.info: %w", err)
	} else {
		fmt.Printf("    Backup: %s\n", bakPath)
	}

	data, err := os.ReadFile(uiTextPath)
	if err != nil {
		return err
	}

	translations, err := loadUITextTranslation(uitextTransPath)
	if err != nil {
		return err
	}
	if len(translations) == 0 {
		fmt.Println("    No translations found — skipping")
		return nil
	}

	patched, err := uitext.PatchEnglish(data, translations)
	if err != nil {
		return err
	}

	// Write to original path (not backup path).
	outPath := strings.TrimSuffix(bakPath, ".bak")
	if err := os.WriteFile(outPath, patched, 0644); err != nil {
		return err
	}
	fmt.Printf("    Patched %d strings in %s\n", len(translations), outPath)
	return nil
}

// loadHintsTranslation reads a hints_swedish.txt file and returns a map of
// address → Swedish text. Only lines with actual translations (not starting
// with [E]) are included.
//
// Format: ADDR<TAB>Swedish text
// Lines starting with # are comments.
func loadHintsTranslation(path string) (map[uint32]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[uint32]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		// Last field is the Swedish text (supports both 2-field and 3-field formats).
		swe := parts[len(parts)-1]
		if strings.HasPrefix(swe, "[E]") {
			continue // untranslated
		}
		addr, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			continue
		}
		result[uint32(addr)] = swe
	}
	return result, scanner.Err()
}

// loadUITextTranslation reads a uitext_swedish.txt file and returns a map of
// key → Swedish text. Only lines with actual translations (not starting with
// [E]) are included.
//
// Format: KEY<TAB>Swedish text
// Lines starting with # are comments.
func loadUITextTranslation(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		key, swe := parts[0], parts[1]
		if strings.HasPrefix(swe, "[E]") {
			continue // untranslated
		}
		result[key] = swe
	}
	return result, scanner.Err()
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
