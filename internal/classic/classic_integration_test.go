//go:build integration

package classic_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"scumm-patcher/internal/classic"
	"scumm-patcher/internal/pak"
)

// repoRoot walks up from the package directory to find the repository root
// (identified as the directory containing go.mod).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod in parent directories")
		}
		dir = parent
	}
}

// integrationPaths returns paths to the resources needed for integration tests.
// Skips the test if any are missing.
func integrationPaths(t *testing.T) (pakPath, scummtrPath string) {
	t.Helper()
	root := repoRoot(t)

	pakPath = filepath.Join(root, "games", "monkey1", "game", "Monkey1.pak")

	switch runtime.GOOS {
	case "linux":
		scummtrPath = filepath.Join(root, "internal/classic/assets/scummtr-linux-x64")
	case "darwin":
		scummtrPath = filepath.Join(root, "internal/classic/assets/scummtr-darwin-x64")
	case "windows":
		scummtrPath = filepath.Join(root, "internal/classic/assets/scummtr-windows-x64.exe")
	default:
		t.Skipf("integration tests not supported on %s", runtime.GOOS)
	}

	missing := []string{}
	for _, p := range []string{pakPath, scummtrPath} {
		if _, err := os.Stat(p); err != nil {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		t.Skipf("integration files missing:\n  %s", strings.Join(missing, "\n  "))
	}
	return
}

// INT-002 (classic): Identity translation — the inject→extract cycle is idempotent.
//
// scummtr normalizes internal string-table structures on first inject, so the
// modified file may not be byte-identical to the original. We accept this and
// instead verify that a second round-trip produces the same result as the first
// (idempotence), meaning scummtr has converged to its canonical format.
func TestIdentityTranslation(t *testing.T) {
	pakPath, scummtrPath := integrationPaths(t)

	// Extract MONKEY1.000 and MONKEY1.001 from the real PAK.
	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	var orig000, orig001 []byte
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			orig000 = append([]byte(nil), e.Data...)
		case "classic/en/monkey1.001":
			orig001 = append([]byte(nil), e.Data...)
		}
	}
	if orig000 == nil || orig001 == nil {
		t.Fatal("classic files not found in PAK")
	}

	// Set up a work directory with uppercase filenames (required by scummtr).
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "MONKEY1.000"), orig000, 0644)
	os.WriteFile(filepath.Join(workDir, "MONKEY1.001"), orig001, 0644)

	// Extract the scummtr binary to a temp location.
	scummtrData, _ := os.ReadFile(scummtrPath)
	scummtrExec := filepath.Join(t.TempDir(), "scummtr")
	os.WriteFile(scummtrExec, scummtrData, 0755)

	stringsFile := filepath.Join(t.TempDir(), "strings.txt")

	// Step 1: Export English strings (identity source).
	runScummtr := func(t *testing.T, label string, args ...string) {
		t.Helper()
		cmd := exec.Command(scummtrExec, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("scummtr %s: %v", label, err)
		}
	}

	runScummtr(t, "export",
		"-g", "monkeycdalt", "-p", workDir, "-c", "-A", "aov", "-o", "-f", stringsFile,
	)

	info, _ := os.Stat(stringsFile)
	if info == nil || info.Size() == 0 {
		t.Fatal("scummtr export produced no output")
	}
	t.Logf("exported %d bytes of strings", info.Size())

	// Step 2: Inject those same strings back (identity).
	runScummtr(t, "inject",
		"-g", "monkeycdalt", "-p", workDir, "-c", "-A", "aov", "-i", "-f", stringsFile,
	)

	mod000, _ := os.ReadFile(filepath.Join(workDir, "MONKEY1.000"))
	mod001, _ := os.ReadFile(filepath.Join(workDir, "MONKEY1.001"))

	if bytes.Equal(mod000, orig000) {
		t.Logf("MONKEY1.000: byte-identical after identity injection (%d bytes)", len(orig000))
	} else {
		t.Logf("MONKEY1.000: normalized by scummtr: %d → %d bytes (expected)", len(orig000), len(mod000))
	}
	if bytes.Equal(mod001, orig001) {
		t.Logf("MONKEY1.001: byte-identical after identity injection (%d bytes)", len(orig001))
	} else {
		t.Logf("MONKEY1.001: normalized by scummtr: %d → %d bytes (expected)", len(orig001), len(mod001))
	}

	// Step 3: Second round-trip — must be idempotent.
	stringsFile2 := filepath.Join(t.TempDir(), "strings2.txt")
	runScummtr(t, "export(2)",
		"-g", "monkeycdalt", "-p", workDir, "-c", "-A", "aov", "-o", "-f", stringsFile2,
	)
	runScummtr(t, "inject(2)",
		"-g", "monkeycdalt", "-p", workDir, "-c", "-A", "aov", "-i", "-f", stringsFile2,
	)

	mod000b, _ := os.ReadFile(filepath.Join(workDir, "MONKEY1.000"))
	mod001b, _ := os.ReadFile(filepath.Join(workDir, "MONKEY1.001"))

	if !bytes.Equal(mod000, mod000b) {
		t.Errorf("MONKEY1.000 not idempotent: 1st=%d bytes, 2nd=%d bytes", len(mod000), len(mod000b))
	} else {
		t.Logf("MONKEY1.000: idempotent (%d bytes)", len(mod000b))
	}
	if !bytes.Equal(mod001, mod001b) {
		t.Errorf("MONKEY1.001 not idempotent: 1st=%d bytes, 2nd=%d bytes", len(mod001), len(mod001b))
	} else {
		t.Logf("MONKEY1.001: idempotent (%d bytes)", len(mod001b))
	}
}

// INT-EXTRACT-PAK: Extracting strings from files sourced from a PAK produces non-empty output.
//
// Mirrors the pipeline used by scripts/extract_pak.sh + scripts/extract_assets.sh:
// classic files are pulled from Monkey1.pak, written to a temp directory with
// uppercase names, then scummtr exports strings from them.
func TestExtractStringsFromPAK(t *testing.T) {
	pakPath, scummtrPath := integrationPaths(t)

	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	var data000, data001 []byte
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			data000 = append([]byte(nil), e.Data...)
		case "classic/en/monkey1.001":
			data001 = append([]byte(nil), e.Data...)
		}
	}
	if data000 == nil || data001 == nil {
		t.Fatal("classic files not found in PAK")
	}

	scummtrData, _ := os.ReadFile(scummtrPath)
	scummtrExec := filepath.Join(t.TempDir(), "scummtr")
	os.WriteFile(scummtrExec, scummtrData, 0755)

	// Write classic files with uppercase names (as the script does).
	classicDir := t.TempDir()
	os.WriteFile(filepath.Join(classicDir, "MONKEY1.000"), data000, 0644)
	os.WriteFile(filepath.Join(classicDir, "MONKEY1.001"), data001, 0644)

	outFile := filepath.Join(t.TempDir(), "strings.txt")
	cmd := exec.Command(scummtrExec,
		"-g", "monkeycdalt", "-p", classicDir, "-cwh", "-A", "aov", "-o", "-f", outFile,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("scummtr export: %v", err)
	}

	info, err := os.Stat(outFile)
	if err != nil || info.Size() == 0 {
		t.Fatal("scummtr export produced no output")
	}
	t.Logf("extracted %d bytes of strings from PAK-sourced files", info.Size())
}

// INT-EXTRACT-DIR: Extracting strings from a directory of classic files produces non-empty output.
//
// Mirrors the directory input mode of scripts/extract_assets.sh:
// the user provides a directory containing MONKEY1.000 + MONKEY1.001 directly,
// skipping the PAK extraction step. Tests both uppercase and lowercase filenames,
// since the script normalises them to uppercase before invoking scummtr.
func TestExtractStringsFromClassicDir(t *testing.T) {
	pakPath, scummtrPath := integrationPaths(t)

	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	var data000, data001 []byte
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			data000 = append([]byte(nil), e.Data...)
		case "classic/en/monkey1.001":
			data001 = append([]byte(nil), e.Data...)
		}
	}
	if data000 == nil || data001 == nil {
		t.Fatal("classic files not found in PAK")
	}

	scummtrData, _ := os.ReadFile(scummtrPath)
	scummtrExec := filepath.Join(t.TempDir(), "scummtr")
	os.WriteFile(scummtrExec, scummtrData, 0755)

	for _, tc := range []struct {
		name    string
		f000    string
		f001    string
	}{
		{"uppercase", "MONKEY1.000", "MONKEY1.001"},
		{"lowercase", "monkey1.000", "monkey1.001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the user's classic files directory.
			inputDir := t.TempDir()
			os.WriteFile(filepath.Join(inputDir, tc.f000), data000, 0644)
			os.WriteFile(filepath.Join(inputDir, tc.f001), data001, 0644)

			// Simulate what the script does: copy to a work dir with uppercase names.
			workDir := t.TempDir()
			os.WriteFile(filepath.Join(workDir, "MONKEY1.000"),
				mustReadFile(t, filepath.Join(inputDir, tc.f000)), 0644)
			os.WriteFile(filepath.Join(workDir, "MONKEY1.001"),
				mustReadFile(t, filepath.Join(inputDir, tc.f001)), 0644)

			outFile := filepath.Join(t.TempDir(), "strings.txt")
			cmd := exec.Command(scummtrExec,
				"-g", "monkeycdalt", "-p", workDir, "-cwh", "-A", "aov", "-o", "-f", outFile,
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("scummtr export: %v", err)
			}

			info, err := os.Stat(outFile)
			if err != nil || info.Size() == 0 {
				t.Fatal("scummtr export produced no output")
			}
			t.Logf("extracted %d bytes of strings from %s directory", info.Size(), tc.name)
		})
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// INT-ROUNDTRIP: InjectTranslation round-trip with English text is idempotent.
//
// This test verifies that our injection pipeline produces correct, non-corrupted
// output by using a language-neutral input: we export the original English strings
// in the exact format InjectTranslation expects (headers, Unix LF, no charset
// conversion), inject them back, re-export in the same format, and assert the
// text is identical to what we started with.
//
// The test uses InjectTranslation directly (not raw scummtr) to catch any bugs
// introduced by our flag choices, encodeForScummtr pre-processing, or temp-file
// handling. Since the English text has no Swedish characters, encodeForScummtr
// is a no-op and any corruption would be directly attributable to our pipeline.
//
// scummtr may normalise the internal string-table structure on first inject, so
// we accept that the game files may not be byte-identical to the originals.
// We verify idempotence: a second inject produces identical re-exported text.
func TestInjectTranslationRoundTrip(t *testing.T) {
	pakPath, scummtrPath := integrationPaths(t)

	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	var orig000, orig001 []byte
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			orig000 = append([]byte(nil), e.Data...)
		case "classic/en/monkey1.001":
			orig001 = append([]byte(nil), e.Data...)
		}
	}
	if orig000 == nil || orig001 == nil {
		t.Fatal("classic files not found in PAK")
	}

	scummtrData, _ := os.ReadFile(scummtrPath)
	scummtrExec := filepath.Join(t.TempDir(), "scummtr")
	os.WriteFile(scummtrExec, scummtrData, 0755)

	exportStrings := func(t *testing.T, gameDir, outFile string) {
		t.Helper()
		// Export with -h (headers) but no -c or -w: produces Unix LF, ASCII+\NNN
		// escapes, with [room:TYPE#resnum] prefixes — exactly the format that
		// InjectTranslation expects on import (-ih).
		cmd := exec.Command(scummtrExec,
			"-g", "monkeycdalt", "-p", gameDir, "-h", "-A", "aov", "-o", "-f", outFile,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("scummtr export: %v", err)
		}
	}

	// Step 1: Export original English strings in InjectTranslation-compatible format.
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "MONKEY1.000"), orig000, 0644)
	os.WriteFile(filepath.Join(workDir, "MONKEY1.001"), orig001, 0644)

	baseline := filepath.Join(t.TempDir(), "baseline.txt")
	exportStrings(t, workDir, baseline)

	baselineData, err := os.ReadFile(baseline)
	if err != nil || len(baselineData) == 0 {
		t.Fatal("baseline export produced no output")
	}
	t.Logf("baseline: %d lines, %d bytes", bytes.Count(baselineData, []byte("\n")), len(baselineData))

	// Step 2: Inject the baseline strings back using our production pipeline.
	if err := classic.InjectTranslation(workDir, baseline); err != nil {
		t.Fatalf("InjectTranslation (first): %v", err)
	}

	// Step 3: Re-export and compare to baseline.
	roundtrip1 := filepath.Join(t.TempDir(), "roundtrip1.txt")
	exportStrings(t, workDir, roundtrip1)

	rt1Data, _ := os.ReadFile(roundtrip1)
	if !bytes.Equal(baselineData, rt1Data) {
		// Show the first differing line to aid diagnosis.
		baseLines := bytes.Split(baselineData, []byte("\n"))
		rt1Lines := bytes.Split(rt1Data, []byte("\n"))
		for i := 0; i < len(baseLines) && i < len(rt1Lines); i++ {
			if !bytes.Equal(baseLines[i], rt1Lines[i]) {
				t.Errorf("first diff at line %d:\n  baseline:  %q\n  roundtrip: %q",
					i+1, baseLines[i], rt1Lines[i])
				break
			}
		}
		if len(baseLines) != len(rt1Lines) {
			t.Errorf("line count: baseline=%d, roundtrip=%d", len(baseLines), len(rt1Lines))
		}
		t.Fatalf("roundtrip text differs from baseline")
	}
	t.Logf("roundtrip 1: text identical to baseline ✓")

	// Step 4: Second inject+export — must still match (idempotent).
	if err := classic.InjectTranslation(workDir, roundtrip1); err != nil {
		t.Fatalf("InjectTranslation (second): %v", err)
	}
	roundtrip2 := filepath.Join(t.TempDir(), "roundtrip2.txt")
	exportStrings(t, workDir, roundtrip2)

	rt2Data, _ := os.ReadFile(roundtrip2)
	if !bytes.Equal(rt1Data, rt2Data) {
		t.Fatalf("inject not idempotent: roundtrip2 differs from roundtrip1")
	}
	t.Logf("roundtrip 2: idempotent ✓")
}

// INT-CLASSIC: InjectTranslation with a real translation file produces a larger .001.
func TestInjectTranslationWithRealFile(t *testing.T) {
	pakPath, _ := integrationPaths(t)
	root := repoRoot(t)
	translationPath := filepath.Join(root, "translation", "monkey1", "swedish.txt")
	if _, err := os.Stat(translationPath); err != nil {
		t.Skipf("translation file not found: %s", translationPath)
	}

	_, _, _, entries, err := pak.Read(pakPath)
	if err != nil {
		t.Fatalf("pak.Read: %v", err)
	}
	var orig000, orig001 []byte
	for _, e := range entries {
		switch strings.ToLower(e.Name) {
		case "classic/en/monkey1.000":
			orig000 = append([]byte(nil), e.Data...)
		case "classic/en/monkey1.001":
			orig001 = append([]byte(nil), e.Data...)
		}
	}

	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "MONKEY1.000"), orig000, 0644)
	os.WriteFile(filepath.Join(workDir, "MONKEY1.001"), orig001, 0644)

	if err := classic.InjectTranslation(workDir, translationPath); err != nil {
		t.Fatalf("InjectTranslation: %v", err)
	}

	mod001, _ := os.ReadFile(filepath.Join(workDir, "MONKEY1.001"))
	if len(mod001) <= len(orig001) {
		t.Errorf("MONKEY1.001 did not grow: orig=%d, patched=%d", len(orig001), len(mod001))
	}
	t.Logf("MONKEY1.001: %d → %d bytes (+%d)", len(orig001), len(mod001), len(mod001)-len(orig001))
}
