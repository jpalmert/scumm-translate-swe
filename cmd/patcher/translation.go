package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// findTranslationFile resolves the translation file path. If explicit is non-empty
// it is used directly. Otherwise swedish.txt is looked up next to the executable.
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
	p := filepath.Join(filepath.Dir(exe), "swedish.txt")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", fmt.Errorf(
		"translation file not found\n"+
			"  Expected: %s\n"+
			"  Or pass the path explicitly as an argument",
		p)
}

// findOptionalSEFile looks for an SE translation file (e.g. "uitext_swedish.txt"
// or "hints_swedish.txt") in the same directory as the base translation file.
// Returns the path if found, or empty string if not.
func findOptionalSEFile(baseTranslation, filename string) string {
	p := filepath.Join(filepath.Dir(baseTranslation), filename)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
