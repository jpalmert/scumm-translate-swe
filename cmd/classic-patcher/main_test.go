package main

import (
	"os"
	"path/filepath"
	"testing"
)

// CLASSIC-001: Non-existent game directory → clear error.
func TestRunClassicPatchMissingDir(t *testing.T) {
	err := runClassicPatch("/nonexistent/game/dir", "/dev/null")
	if err == nil {
		t.Fatal("expected error for missing game dir")
	}
}

// CLASSIC-002: Directory missing MONKEY1.000 → clear error.
func TestRunClassicPatchMissing000(t *testing.T) {
	dir := t.TempDir()
	// Only create .001, not .000
	os.WriteFile(filepath.Join(dir, "MONKEY1.001"), []byte("data"), 0644)

	err := runClassicPatch(dir, "/dev/null")
	if err == nil {
		t.Fatal("expected error for missing MONKEY1.000")
	}
}

// CLASSIC-003: Directory missing MONKEY1.001 → clear error.
func TestRunClassicPatchMissing001(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), []byte("data"), 0644)
	// no .001

	err := runClassicPatch(dir, "/dev/null")
	if err == nil {
		t.Fatal("expected error for missing MONKEY1.001")
	}
}

// CLASSIC-004: Translation file not found → clear error (even if game files are present).
func TestRunClassicPatchMissingTranslation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(dir, "MONKEY1.001"), []byte("data"), 0644)

	err := runClassicPatch(dir, "/nonexistent/monkey1_swe.txt")
	if err == nil {
		t.Fatal("expected error for missing translation file")
	}
}

// CLASSIC-005: Lowercase filenames are accepted (Linux game installations).
func TestFindGameFileLowercase(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "monkey1.000"), []byte("lower"), 0644)
	os.WriteFile(filepath.Join(dir, "monkey1.001"), []byte("lower"), 0644)

	p, err := findGameFile(dir, "MONKEY1.000", "monkey1.000")
	if err != nil {
		t.Fatalf("findGameFile: %v", err)
	}
	if filepath.Base(p) != "monkey1.000" {
		t.Errorf("expected lowercase path, got %s", p)
	}
}

// CLASSIC-006: Uppercase filenames are preferred over lowercase when both exist.
func TestFindGameFileUppercasePreferred(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MONKEY1.000"), []byte("upper"), 0644)
	os.WriteFile(filepath.Join(dir, "monkey1.000"), []byte("lower"), 0644)

	p, err := findGameFile(dir, "MONKEY1.000", "monkey1.000")
	if err != nil {
		t.Fatalf("findGameFile: %v", err)
	}
	if filepath.Base(p) != "MONKEY1.000" {
		t.Errorf("expected uppercase path, got %s", p)
	}
}

// CLASSIC-007: findGameFile returns error when neither name exists.
func TestFindGameFileMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := findGameFile(dir, "MONKEY1.000", "monkey1.000")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// CLASSIC-008: findTranslationFile returns error for missing explicit path.
func TestFindTranslationFileMissingExplicit(t *testing.T) {
	_, err := findTranslationFile("/nonexistent/translation.txt")
	if err == nil {
		t.Fatal("expected error")
	}
}

// CLASSIC-009: findTranslationFile accepts a valid explicit path.
func TestFindTranslationFileExplicit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "monkey1_swe.txt")
	os.WriteFile(p, []byte("translation data"), 0644)

	got, err := findTranslationFile(p)
	if err != nil {
		t.Fatalf("findTranslationFile: %v", err)
	}
	if got != p {
		t.Errorf("got %q, want %q", got, p)
	}
}
