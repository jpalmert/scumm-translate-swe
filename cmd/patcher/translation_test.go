package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"
)

// repoRoot returns the repository root by walking up from the test file location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// cmd/patcher/translation_test.go → repo root is two levels up
	return filepath.Join(filepath.Dir(f), "..", "..")
}

var tagRe = regexp.MustCompile(`^\[.*?\]\((.{2})\)(.*)`)

// TestFADialogChoiceLength verifies that no translated (FA) dialog choice line
// exceeds 57 characters per visual segment. The SCUMM dialog selection box
// is ~58 characters wide; lines longer than 57 get clipped.
//
// Lines may contain \254\NNN line-break control codes that split the text
// across multiple rows in the selection box. Each visual segment is measured
// independently.
func TestFADialogChoiceLength(t *testing.T) {
	root := repoRoot(t)
	txPath := filepath.Join(root, "games", "monkey1", "translation", "swedish.txt")

	f, err := os.Open(txPath)
	if err != nil {
		t.Skipf("translation file not found: %v", err)
	}
	defer f.Close()

	const maxLen = 57
	lineBreak := regexp.MustCompile(`\\254\\[0-9]{3}`)

	scanner := bufio.NewScanner(f)
	lineNum := 0
	var violations []string
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		m := tagRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		opcode, text := m[1], m[2]
		if opcode != "FA" {
			continue
		}
		// Skip untranslated lines (still prefixed with [E])
		if strings.HasPrefix(text, "[E]") {
			continue
		}

		segments := lineBreak.Split(text, -1)
		for _, seg := range segments {
			seg = strings.TrimSpace(seg)
			n := utf8.RuneCountInString(seg)
			if n > maxLen {
				violations = append(violations, fmt.Sprintf(
					"line %d (%d chars): %s", lineNum, n, seg))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading translation file: %v", err)
	}

	for _, v := range violations {
		t.Errorf("FA line too long (max %d): %s", maxLen, v)
	}
}
