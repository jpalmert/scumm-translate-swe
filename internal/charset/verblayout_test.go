package charset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildVerbPattern constructs a synthetic byte sequence matching the pattern
// that findVerbXOffset searches for:
//
//	funcCode 0x09 0x02 <label> 0x00 0x13 0x12 <shortcut> 0x05 <X> 0x00 <Y>
func buildVerbPattern(funcCode, x, y byte, label string) []byte {
	var b []byte
	b = append(b, funcCode, 0x09, 0x02)
	b = append(b, []byte(label)...)
	b = append(b, 0x00)       // null terminator
	b = append(b, 0x13, 0x12) // post-label markers
	b = append(b, 0x41)       // shortcut key (arbitrary)
	b = append(b, 0x05)       // end marker
	b = append(b, x, 0x00, y) // X, padding, Y
	return b
}

func TestFindVerbXOffset(t *testing.T) {
	t.Run("normal match", func(t *testing.T) {
		padding := []byte{0xFF, 0xFF, 0xFF}
		pattern := buildVerbPattern(0x04, 0x16, 0x9B, "Give")
		data := append(padding, pattern...)

		off, err := findVerbXOffset(data, 0x04)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data[off] != 0x16 {
			t.Errorf("X byte at offset %d = 0x%02X, want 0x16", off, data[off])
		}
		if data[off+2] != 0x9B {
			t.Errorf("Y byte at offset %d = 0x%02X, want 0x9B", off+2, data[off+2])
		}
	})

	t.Run("not found", func(t *testing.T) {
		data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
		_, err := findVerbXOffset(data, 0x04)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want it to contain 'not found'", err.Error())
		}
	})

	t.Run("ambiguous duplicate pattern", func(t *testing.T) {
		p1 := buildVerbPattern(0x04, 0x16, 0x9B, "Give")
		p2 := buildVerbPattern(0x04, 0x48, 0xAB, "Give")
		data := append(p1, p2...)

		_, err := findVerbXOffset(data, 0x04)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "ambiguous") {
			t.Errorf("error = %q, want it to contain 'ambiguous'", err.Error())
		}
	})
}

func TestPatchVerbCoords(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// Build synthetic SCRP data containing all 9 verb patterns.
		var data []byte
		for _, v := range verbLayout {
			// Use placeholder original coordinates that differ from the target.
			pattern := buildVerbPattern(v.funcCode, 0x00, 0x00, v.name)
			data = append(data, pattern...)
		}

		patched, err := patchVerbCoords(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify each verb got its new coordinates.
		for _, v := range verbLayout {
			off, err := findVerbXOffset(patched, v.funcCode)
			if err != nil {
				t.Fatalf("verb %q: %v", v.name, err)
			}
			if patched[off] != v.newX {
				t.Errorf("verb %q X = 0x%02X, want 0x%02X", v.name, patched[off], v.newX)
			}
			if patched[off+2] != v.newY {
				t.Errorf("verb %q Y = 0x%02X, want 0x%02X", v.name, patched[off+2], v.newY)
			}
		}
	})

	t.Run("missing verb entry", func(t *testing.T) {
		// Provide only 8 of 9 verbs — omit the first one (funcCode 0x04 "Give").
		var data []byte
		for _, v := range verbLayout[1:] {
			pattern := buildVerbPattern(v.funcCode, 0x00, 0x00, v.name)
			data = append(data, pattern...)
		}

		_, err := patchVerbCoords(data)
		if err == nil {
			t.Fatal("expected error for missing verb, got nil")
		}
		if !strings.Contains(err.Error(), "Give") {
			t.Errorf("error = %q, want it to mention 'Give'", err.Error())
		}
	})
}

func TestFindFileInTree(t *testing.T) {
	t.Run("file at root level", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "SCRP_0022")
		if err := os.WriteFile(target, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := findFileInTree(dir, "SCRP_0022")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != target {
			t.Errorf("got %q, want %q", got, target)
		}
	})

	t.Run("file in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "DISK_0001", "LECF", "LFLF_0010")
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(sub, "SCRP_0022")
		if err := os.WriteFile(target, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := findFileInTree(dir, "SCRP_0022")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != target {
			t.Errorf("got %q, want %q", got, target)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		dir := t.TempDir()

		_, err := findFileInTree(dir, "SCRP_0022")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want it to contain 'not found'", err.Error())
		}
	})
}
