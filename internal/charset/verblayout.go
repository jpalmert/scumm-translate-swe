package charset

// PatchVerbLayout repositions the nine verb action buttons in MONKEY1.001 to
// a layout better suited to the Swedish word lengths.
//
// # Verb button layout
//
// In SCUMM v5 each verb button is defined in script SCRP_0022 with both its
// action ID and its screen coordinates. Changing the coordinates moves the button
// to a different cell in the 3×3 grid — the action travels with it.
//
// Original English layout:
//
//	Left (x=0x16) | Middle (x=0x48) | Right (x=0x7C)
//	Give          | Pick up          | Use             ← Top    (y=0x9B)
//	Open          | Look at          | Push            ← Middle (y=0xAB)
//	Close         | Talk to          | Pull            ← Bottom (y=0xBB)
//
// Patched layout (Swedish — shorter words on the right):
//
//	Left (x=0x16) | Middle (x=0x48) | Right (x=0x7C)
//	Öppna         | Titta            | Ge              ← Top    (y=0x9B)
//	Stäng         | Tala             | Ta              ← Middle (y=0xAB)
//	Putta         | Använd           | Dra             ← Bottom (y=0xBB)
//
// # How the patch is applied
//
// The coordinate patch is applied at runtime, AFTER scummtr has injected Swedish
// verb labels into SCRP_0022. This preserves the Swedish text while moving the
// buttons to their new grid positions. PatchVerbLayout uses scummrp to:
//  1. Dump all SCRP blocks from the game directory.
//  2. Patch ONLY the X/Y coordinates in SCRP_0022, leaving the verb labels intact.
//  3. Reimport the SCRP blocks back into MONKEY1.001.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// verbEntry describes how to reposition one verb action button.
type verbEntry struct {
	funcCode byte
	name     string // English name for error messages only
	newX     byte
	newY     byte
}

// verbLayout maps each verb's action ID to its new grid position.
// Shorter Swedish words are placed on the right; longer ones on the left.
var verbLayout = []verbEntry{
	{0x04, "Give", 0x7C, 0x9B},    // Left/Top     → Right/Top
	{0x02, "Open", 0x16, 0x9B},    // Left/Middle  → Left/Top
	{0x03, "Close", 0x16, 0xAB},   // Left/Bottom  → Left/Middle
	{0x09, "Pick up", 0x7C, 0xAB}, // Mid/Top      → Right/Middle
	{0x08, "Look at", 0x48, 0x9B}, // Mid/Middle   → Mid/Top
	{0x0A, "Talk to", 0x48, 0xAB}, // Mid/Bottom   → Mid/Middle
	{0x07, "Use", 0x48, 0xBB},     // Right/Top    → Mid/Bottom
	{0x05, "Push", 0x16, 0xBB},    // Right/Middle → Left/Bottom
	{0x06, "Pull", 0x7C, 0xBB},    // Right/Bottom → unchanged
}

// PatchVerbLayout patches verb button positions in the game files in gameDir.
// gameDir must contain MONKEY1.000 and MONKEY1.001 in uppercase.
//
// The coordinates in SCRP_0022 are patched in-memory so that the verb label
// strings (which may already be in Swedish from a prior scummtr injection) are
// preserved unchanged.
func PatchVerbLayout(gameDir string) error {
	env, err := setupScummrp("scummrp-verbs-*")
	if err != nil {
		return err
	}
	defer env.cleanup()

	// Dump all SCRP blocks from MONKEY1.001.
	if err := env.run("-g", "monkeycdalt", "-p", gameDir, "-t", "SCRP", "-od", env.dumpDir); err != nil {
		return fmt.Errorf("scummrp export SCRP: %w", err)
	}

	// Locate SCRP_0022 in the dump — path differs between game variants.
	scrpPath, err := findFileInTree(env.dumpDir, "SCRP_0022")
	if err != nil {
		return fmt.Errorf("SCRP_0022 not found in scummrp dump: %w", err)
	}

	// Read the current SCRP_0022 (may already contain Swedish labels from scummtr).
	scrpData, err := os.ReadFile(scrpPath)
	if err != nil {
		return fmt.Errorf("read SCRP_0022: %w", err)
	}

	// Patch only the X/Y coordinates, preserving whatever label text is present.
	patched, err := patchVerbCoords(scrpData)
	if err != nil {
		return fmt.Errorf("patch SCRP_0022 coordinates: %w", err)
	}

	if err := os.WriteFile(scrpPath, patched, 0644); err != nil {
		return fmt.Errorf("write SCRP_0022: %w", err)
	}

	// Reimport the patched SCRP blocks back into MONKEY1.001.
	if err := env.run("-g", "monkeycdalt", "-p", gameDir, "-t", "SCRP", "-id", env.dumpDir); err != nil {
		return fmt.Errorf("scummrp import SCRP: %w", err)
	}

	return nil
}

// patchVerbCoords patches the grid X/Y coordinates for each verb in SCRP_0022
// without modifying the verb label strings.
func patchVerbCoords(data []byte) ([]byte, error) {
	result := make([]byte, len(data))
	copy(result, data)

	for _, v := range verbLayout {
		xOff, err := findVerbXOffset(result, v.funcCode)
		if err != nil {
			return nil, fmt.Errorf("verb %q (0x%02X): %w", v.name, v.funcCode, err)
		}
		result[xOff] = v.newX
		result[xOff+2] = v.newY
	}
	return result, nil
}

// findVerbXOffset locates the X coordinate byte for a verb entry in SCRP_0022.
//
// Pattern searched:
//
//	funcCode 0x09 0x02 <label bytes> 0x00 0x13 0x12 <shortcut> 0x05 <X> 0x00 <Y>
//
// The label may contain any non-null bytes (ASCII or SCUMM-encoded Swedish chars).
// Returns the byte offset of X (Y is at offset+2).
func findVerbXOffset(data []byte, funcCode byte) (int, error) {
	var candidates []int
	for i := 0; i < len(data)-8; i++ {
		if data[i] != funcCode || data[i+1] != 0x09 || data[i+2] != 0x02 {
			continue
		}
		// Scan past the label to the null terminator.
		j := i + 3
		for j < len(data) && data[j] != 0x00 {
			j++
		}
		if j >= len(data) {
			continue
		}
		// After null: expect 0x13 0x12 <shortcut> 0x05
		if j+4 < len(data) && data[j+1] == 0x13 && data[j+2] == 0x12 && data[j+4] == 0x05 {
			candidates = append(candidates, j+5) // offset of X byte
		}
	}
	if len(candidates) == 0 {
		return 0, fmt.Errorf("not found")
	}
	if len(candidates) > 1 {
		return 0, fmt.Errorf("ambiguous: %d matches", len(candidates))
	}
	return candidates[0], nil
}

// findFileInTree walks root and returns the path of the first file named target.
func findFileInTree(root, target string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == target {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("%s not found under %s", target, root)
	}
	return found, nil
}
