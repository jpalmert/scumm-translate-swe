// MI1 Swedish Translation Patcher
//
// Patches The Secret of Monkey Island with the Swedish translation.
// Works with both the Special Edition (Monkey1.pak) and the Classic CD-ROM version
// (MONKEY1.000 / MONKEY1.001). The version is detected automatically.
//
// # Architecture overview
//
// The patcher is a single self-contained binary. All tools and glyph data are
// embedded at compile time via //go:embed so the user only needs the binary and
// swedish.txt (the translation file).
//
// Internal packages:
//
//	internal/pak     — Read and write the MI1SE PAK archive format.
//	                   Used only by the SE pipeline. Handles both Steam (LPAK) and
//	                   GOG (KAPL) magic bytes.
//
//	internal/classic — Inject Swedish text into MONKEY1.000/001 using scummtr.
//	                   Embeds platform-specific scummtr binaries (Linux/macOS/Windows).
//	                   Swedish UTF-8 characters are pre-encoded to SCUMM escape codes
//	                   (e.g. å→\123) because scummtr's -c flag is unreliable for
//	                   the monkeycdalt game ID.
//
//	internal/charset — Patch the five classic CHAR blocks with Swedish glyph bitmaps
//	                   using scummrp. Covers all on-screen fonts (see charset.go for
//	                   per-block details). Needed for Classic mode (F1 toggle in SE).
//	                   Embeds platform-specific scummrp binaries.
//
//	internal/font    — Patch the glyph lookup table in SE .font files so that SCUMM
//	                   internal codes (91–93, 123–125, 130) resolve to the Swedish
//	                   glyphs already present in the .font glyph atlas. This is a
//	                   pure lookup-table patch — no new glyph images are added.
//	                   Needed for SE mode rendering.
//
//	internal/backup  — Create .bak safety copies of files before they are overwritten.
//
// Build pipeline (scripts/build.sh):
//  1. Download scummtr binaries (internal/classic/assets/)         — install_deps.sh / build.sh
//  2. Generate patched CHAR .bin files (internal/charset/assets/)  — build_char_assets.sh
//  3. Cross-compile for Linux, macOS, Windows (dist/)
//
// # Simple usage
//
// Place the patcher and swedish.txt next to your game files and run:
//
//	mi1-translate-linux
//
// # Advanced usage
//
//	mi1-translate <Monkey1.pak> [output.pak] [swedish.txt]   (SE version)
//	mi1-translate <game_dir>    [swedish.txt]                (Classic version)
//
// After patching, start a new game. Swedish text replaces the English strings directly.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	// Parse arguments.
	// --list → dump PAK entry names and exit.
	// .txt extension → translation file; .pak extension → SE input; anything else → output path or game dir.
	listMode := false
	inputPath := ""
	outputPAK := ""
	translationArg := ""
	for _, arg := range os.Args[1:] {
		lower := strings.ToLower(arg)
		switch {
		case lower == "--list":
			listMode = true
		case strings.HasSuffix(lower, ".txt"):
			translationArg = arg
		case inputPath == "":
			inputPath = arg
		default:
			outputPAK = arg
		}
	}

	// No input path given — auto-detect from the executable's directory.
	if inputPath == "" {
		var err error
		inputPath, err = autoDetect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
			printUsage()
			os.Exit(1)
		}
	}

	// --list: dump all entry names from the PAK and exit.
	if listMode {
		if err := runListPAK(inputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Determine mode and run.
	var runErr error
	if isSEInput(inputPath) {
		fmt.Printf("Mode: Special Edition\n")
		runErr = runSEPatch(inputPath, outputPAK, translationArg)
	} else {
		fmt.Printf("Mode: Classic (ScummVM)\n")
		runErr = runClassicPatch(inputPath, translationArg)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", runErr)
		pauseIfWindows()
		os.Exit(1)
	}

	fmt.Println("\nDone!")
	fmt.Println("Start a new game to see Swedish text.")
}

// autoDetect looks for game files next to the executable and returns the input path.
func autoDetect() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}
	exeDir := filepath.Dir(exe)

	// SE: Monkey1.pak
	if _, err := os.Stat(filepath.Join(exeDir, "Monkey1.pak")); err == nil {
		return filepath.Join(exeDir, "Monkey1.pak"), nil
	}

	// Classic: MONKEY1.000 / monkey1.000 / MONKEY.000 / monkey.000
	for _, name := range []string{"MONKEY1.000", "monkey1.000", "MONKEY.000", "monkey.000"} {
		if _, err := os.Stat(filepath.Join(exeDir, name)); err == nil {
			return exeDir, nil
		}
	}

	return "", fmt.Errorf(
		"no game files found next to this executable\n" +
			"  Expected: Monkey1.pak  (Special Edition)\n" +
			"  Or:       MONKEY1.000  (Classic CD-ROM)\n" +
			"  Or:       MONKEY.000   (Classic CD-ROM alternate naming)")
}

// isSEInput returns true if the path looks like a PAK file (SE), false if a directory (Classic).
func isSEInput(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// Doesn't exist yet — go by extension (e.g. explicit output path for SE).
		return strings.HasSuffix(strings.ToLower(path), ".pak")
	}
	if info.IsDir() {
		return false
	}
	return true // existing file → treat as PAK
}

// pauseIfWindows prints a prompt and waits for Enter on Windows, so the CMD
// window opened by a double-click stays visible long enough to read the error.
func pauseIfWindows() {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n') //nolint:errcheck
	}
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "MI1 Swedish Translation Patcher\n\n")
	fmt.Fprintf(os.Stderr, "Simple: place %s and swedish.txt next to your game files and run it.\n\n", exe)
	fmt.Fprintf(os.Stderr, "Advanced:\n")
	fmt.Fprintf(os.Stderr, "  SE:      %s <Monkey1.pak> [output.pak] [swedish.txt]\n", exe)
	fmt.Fprintf(os.Stderr, "  Classic: %s <game_dir> [swedish.txt]\n\n", exe)
	fmt.Fprintf(os.Stderr, "After patching, start a new game to see Swedish text.\n")
}
