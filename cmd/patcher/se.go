package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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

	// --- Step 8: Repack PAK ---
	fmt.Println("\n==> Repacking PAK...")
	if err := pak.Write(outputPAK, hdr, indexBlob, namesBlob, entries); err != nil {
		return fmt.Errorf("writing PAK: %w", err)
	}
	fmt.Printf("    Written: %s\n", outputPAK)

	// --- Step 9: Patch SE engine binary ---
	// Two patches are applied to MISE.exe:
	//
	//  a) Autosave timer: raise the 5-minute threshold to 9,999,999 s to prevent
	//     autosave from firing (autosave triggers a load which crashes with a
	//     modified MONKEY1.001 — see note b).
	//
	//  b) Save/load crash fix: the SE engine's save-file resource-restore loop
	//     (FUN_0049ab60) allocates heap space for each saved resource block. After
	//     scummtr raw-mode injection, LFLF blocks are larger (Swedish strings), so
	//     the combined allocation size exceeds the fixed SCUMM heap capacity while
	//     type-0xc/2/10 resources are locked in place. The allocator returns NULL,
	//     FUN_00499050 writes to address 4, and the engine crashes. The fix patches
	//     FUN_0049ab60 to return 0 immediately (skip resource data restore); script
	//     PCs are already saved by FUN_0049b3e0's variable-size state block
	//     (DAT_005c4680 is within DAT_005c4460's range), and resources reload from
	//     the patched MONKEY1.001 on demand — no save/load regression.
	//
	// The engine binary is optional — patching is skipped if not found.
	fmt.Println("\n==> Patching engine binary (autosave timer + save/load crash fix)...")
	gameDir := filepath.Dir(outputPAK)
	enginePatched := false
	for _, exeName := range []string{"MISE.exe", "mise.exe"} {
		exePath := filepath.Join(gameDir, exeName)
		if _, err := os.Stat(exePath); err != nil {
			continue
		}
		if err := patchAutosaveTimer(exePath); err != nil {
			fmt.Printf("    WARNING (autosave): %v\n", err)
		}
		if err := patchSaveLoadCrash(exePath); err != nil {
			fmt.Printf("    WARNING (save/load): %v\n", err)
		}
		enginePatched = true
		break
	}
	if !enginePatched {
		fmt.Println("    MISE.exe not found — skipping (engine binary not in same directory as PAK)")
	}

	return nil
}

// patchAutosaveTimer raises the 5-minute autosave timer in the SE engine binary
// to 9,999,999 seconds, effectively disabling autosave. The timer is stored as
// an IEEE 754 double (300.0) at a known offset; the patch is skipped with a
// warning if the bytes don't match (different engine version).
func patchAutosaveTimer(exePath string) error {
	const timerOffset = 0xed010

	data, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Base(exePath), err)
	}
	if timerOffset+8 > len(data) {
		return fmt.Errorf("%s: file too small — may be a different version", filepath.Base(exePath))
	}

	expectedBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(expectedBytes, math.Float64bits(300.0))

	// If already patched, read from backup to get original bytes for re-patching.
	bakPath := exePath + ".bak"
	_, bakErr := os.Stat(bakPath)
	if bakErr == nil {
		fmt.Printf("    %s: backup exists — re-patching from original\n", filepath.Base(exePath))
		origData, err := os.ReadFile(bakPath)
		if err != nil {
			return fmt.Errorf("reading backup: %w", err)
		}
		data = origData
	}

	if !bytes.Equal(data[timerOffset:timerOffset+8], expectedBytes) {
		return fmt.Errorf("%s: timer bytes don't match 300.0 at offset 0x%x — skipping (different version?)",
			filepath.Base(exePath), timerOffset)
	}

	if bakErr != nil {
		// No backup yet — create one.
		if _, err := backup.Create(exePath); err != nil {
			return fmt.Errorf("backup %s: %w", filepath.Base(exePath), err)
		}
		fmt.Printf("    Backed up: %s\n", bakPath)
	}

	patched := make([]byte, len(data))
	copy(patched, data)
	binary.LittleEndian.PutUint64(patched[timerOffset:], math.Float64bits(9999999.0))
	if err := os.WriteFile(exePath, patched, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(exePath), err)
	}
	fmt.Printf("    %s: autosave timer patched (300s → 9999999s)\n", filepath.Base(exePath))
	return nil
}

// patchSaveLoadCrash fixes a heap-overflow crash that occurs when loading a save
// game after scummtr raw-mode injection has enlarged MONKEY1.001 resources.
//
// Root cause: the SE engine's resource-restore loop (FUN_0049ab60) allocates heap
// memory for every saved resource block. Raw-mode injection makes LFLF blocks
// larger (Swedish strings are longer than English). When loading, types 0xc/2/10
// remain locked in the fixed SCUMM heap; the remaining free space is no longer
// sufficient to load all larger resource blocks simultaneously. FUN_004990e0
// returns 0 (allocation failure), FUN_00499050 writes to address 4, crash.
//
// Fix: patch the first 3 bytes of FUN_0049ab60 to "XOR EAX,EAX; RET" (31 C0 C3).
// This makes the restore loop exit immediately — resources reload from the patched
// MONKEY1.001 on demand instead. Script PCs are preserved by FUN_0049b3e0's
// variable-size state block (DAT_005c4680 is within DAT_005c4460's range).
//
// Expected first byte at offset 0x9ab60: 0x53 (PUSH EBX, start of function).
// If already patched (first byte 0x31), this is a no-op.
func patchSaveLoadCrash(exePath string) error {
	const funcOffset = 0x9ab60 // file offset of FUN_0049ab60 in MISE.exe

	data, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Base(exePath), err)
	}
	if funcOffset+3 > len(data) {
		return fmt.Errorf("%s: file too small — may be a different version", filepath.Base(exePath))
	}

	// 31 C0 C3 = XOR EAX,EAX; RET
	patchBytes := []byte{0x31, 0xC0, 0xC3}

	switch data[funcOffset] {
	case 0x31:
		// Already patched — idempotent.
		fmt.Printf("    %s: save/load crash fix already applied\n", filepath.Base(exePath))
		return nil
	case 0x53:
		// Expected: PUSH EBX (function prologue). Apply patch.
	default:
		return fmt.Errorf("%s: unexpected byte 0x%02x at offset 0x%x — skipping (different version?)",
			filepath.Base(exePath), data[funcOffset], funcOffset)
	}

	patched := make([]byte, len(data))
	copy(patched, data)
	copy(patched[funcOffset:], patchBytes)
	if err := os.WriteFile(exePath, patched, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(exePath), err)
	}
	fmt.Printf("    %s: save/load crash fix applied (FUN_0049ab60 → XOR EAX,EAX; RET)\n", filepath.Base(exePath))
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
