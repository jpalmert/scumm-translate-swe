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

	pakPath = filepath.Join(root, "game", "monkey1", "Monkey1.pak")

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

// INT-CLASSIC: InjectTranslation with a real translation file produces a larger .001.
func TestInjectTranslationWithRealFile(t *testing.T) {
	pakPath, _ := integrationPaths(t)
	root := repoRoot(t)
	translationPath := filepath.Join(root, "translation", "monkey1", "monkey1_swe.txt")
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
