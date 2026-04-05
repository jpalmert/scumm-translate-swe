package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// findTranslationFile resolves the translation file path. If explicit is non-empty
// it is used directly. Otherwise monkey1.txt is looked up next to the executable.
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
	p := filepath.Join(filepath.Dir(exe), "monkey1.txt")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", fmt.Errorf(
		"translation file not found\n"+
			"  Expected: %s\n"+
			"  Or pass the path explicitly as an argument",
		p)
}
