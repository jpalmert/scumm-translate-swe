// MI1SE Swedish Translation Patcher
//
// Patches Monkey1.pak (Monkey Island 1 Special Edition) with the Swedish translation.
//
// Simple usage — place the patcher and monkey1_swe.txt next to Monkey1.pak and run:
//
//	se-patcher-linux
//
// The patcher will find Monkey1.pak automatically, patch it in-place, and create
// Monkey1.pak.bak as a backup.
//
// Advanced usage:
//
//	se-patcher <Monkey1.pak> [output.pak] [translation_file]
//
//	Monkey1.pak       Path to your original GOG or Steam game file.
//	output.pak        Where to write the patched file. If omitted, patches in place
//	                  and creates Monkey1.pak.bak before overwriting.
//	translation_file  Path to monkey1_swe.txt (default: next to this executable).
//
// After patching, set the in-game language to French to see the Swedish text.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"scumm-patcher/internal/backup"
	"scumm-patcher/internal/classic"
	"scumm-patcher/internal/font"
	"scumm-patcher/internal/pak"
)

func main() {
	// Parse arguments.
	// Convention: a .txt extension = translation file; anything else = PAK/output path.
	inputPAK := ""
	outputPAK := ""
	translationArg := ""
	for _, arg := range os.Args[1:] {
		if strings.HasSuffix(strings.ToLower(arg), ".txt") {
			translationArg = arg
		} else if inputPAK == "" {
			inputPAK = arg
		} else {
			outputPAK = arg
		}
	}

	// No PAK specified — look for Monkey1.pak next to the executable.
	if inputPAK == "" {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine executable path: %v\n", err)
			os.Exit(1)
		}
		candidate := filepath.Join(filepath.Dir(exe), "Monkey1.pak")
		if _, err := os.Stat(candidate); err != nil {
			printUsage()
			os.Exit(1)
		}
		inputPAK = candidate
	}

	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Input:    %s\n", inputPAK)
	if outputPAK == "" {
		fmt.Printf("Output:   %s (in-place with backup)\n\n", inputPAK)
	} else {
		fmt.Printf("Output:   %s\n\n", outputPAK)
	}

	if err := runSEPatch(inputPAK, outputPAK, translationArg); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nDone!")
	fmt.Println("Replace your Monkey1.pak with the output file,")
	fmt.Println("then set the in-game language to French.")
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "MI1SE Swedish Translation Patcher\n\n")
	fmt.Fprintf(os.Stderr, "Simple usage: place %s and monkey1_swe.txt next to Monkey1.pak and run it.\n\n", exe)
	fmt.Fprintf(os.Stderr, "Advanced usage: %s <Monkey1.pak> [output.pak] [translation_file]\n\n", exe)
	fmt.Fprintf(os.Stderr, "  Monkey1.pak       Path to your original GOG/Steam game file\n")
	fmt.Fprintf(os.Stderr, "  output.pak        Output path (default: patch Monkey1.pak in-place)\n")
	fmt.Fprintf(os.Stderr, "  translation_file  Path to monkey1_swe.txt (.txt extension required)\n")
	fmt.Fprintf(os.Stderr, "                    (default: monkey1_swe.txt next to this executable)\n\n")
	fmt.Fprintf(os.Stderr, "In-place mode creates Monkey1.pak.bak before overwriting.\n")
	fmt.Fprintf(os.Stderr, "After patching, set the in-game language to French.\n")
}

// runSEPatch is the testable entry point for the SE patching pipeline.
//
// outputPAK: if empty, patches inputPAK in-place (with backup).
// translationArg: if empty, monkey1_swe.txt is looked up next to the executable.
func runSEPatch(inputPAK, outputPAK, translationArg string) error {
	// Resolve translation file.
	translationPath, err := findTranslationFile(translationArg)
	if err != nil {
		return err
	}
	fmt.Printf("Translation: %s\n\n", translationPath)

	// Determine output path and whether to backup.
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

	// Find the embedded classic SCUMM files.
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
	tmpDir, err := os.MkdirTemp("", "mi1se-patcher-*")
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

	entry000.Data = patched000
	entry001.Data = patched001

	// --- Step 6: Patch font lookup tables ---
	// The SE font files already contain Swedish glyphs at Windows-1252 positions,
	// but the text was injected using SCUMM internal codes (91=Å, 123=å, etc.).
	// Remap those positions to point at the correct existing glyphs.
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

	return nil
}

// remapFontEntries patches the glyph lookup table in every .font entry so that
// SCUMM internal character codes point to the correct Swedish glyphs.
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
			"  Or pass the path explicitly: %s <Monkey1.pak> [output.pak] <translation.txt>",
		p, filepath.Base(os.Args[0]))
}
