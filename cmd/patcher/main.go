// MI1 Swedish Translation Patcher
//
// Patches The Secret of Monkey Island with the Swedish translation.
// Works with both the Special Edition (Monkey1.pak) and the Classic CD-ROM version
// (MONKEY1.000 / MONKEY1.001). The version is detected automatically.
//
// Simple usage — place the patcher and monkey1.txt next to your game files and run:
//
//	mi1-patcher-linux
//
// Advanced usage:
//
//	mi1-patcher <Monkey1.pak> [output.pak] [monkey1.txt]   (SE version)
//	mi1-patcher <game_dir>    [monkey1.txt]                (Classic version)
//
// After patching, set the in-game language to French to see the Swedish text.
// For the Classic version, use ScummVM.
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
	// .txt extension → translation file; .pak extension → SE input; anything else → output path or game dir.
	inputPath := ""
	outputPAK := ""
	translationArg := ""
	for _, arg := range os.Args[1:] {
		lower := strings.ToLower(arg)
		switch {
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
	fmt.Println("Set the in-game language to French to see Swedish text.")
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

	// Classic: MONKEY1.000 or monkey1.000
	for _, name := range []string{"MONKEY1.000", "monkey1.000"} {
		if _, err := os.Stat(filepath.Join(exeDir, name)); err == nil {
			return exeDir, nil
		}
	}

	return "", fmt.Errorf(
		"no game files found next to this executable\n" +
			"  Expected: Monkey1.pak  (Special Edition)\n" +
			"  Or:       MONKEY1.000  (Classic CD-ROM)")
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
	fmt.Fprintf(os.Stderr, "Simple: place %s and monkey1.txt next to your game files and run it.\n\n", exe)
	fmt.Fprintf(os.Stderr, "Advanced:\n")
	fmt.Fprintf(os.Stderr, "  SE:      %s <Monkey1.pak> [output.pak] [monkey1.txt]\n", exe)
	fmt.Fprintf(os.Stderr, "  Classic: %s <game_dir> [monkey1.txt]\n\n", exe)
	fmt.Fprintf(os.Stderr, "After patching, set the in-game language to French.\n")
}
