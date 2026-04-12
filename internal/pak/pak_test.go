package pak_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"scumm-patcher/internal/pak"
)

// buildSyntheticPAK creates a minimal, valid PAK file in memory.
//
// Layout:
//   - Header (40 bytes)
//   - Index  (4 bytes, all zeros)
//   - Entries (numFiles × 20 bytes)
//   - Names  (null-terminated filenames, sequential)
//   - Data   (file contents, sequential)
func buildSyntheticPAK(t *testing.T, magic [4]byte, files []struct{ name, data string }) []byte {
	t.Helper()

	const headerSize = 40
	const indexSize = 4
	const entrySize = uint32(20)
	numFiles := uint32(len(files))

	// Build names blob
	var namesBlob []byte
	namePosMap := make([]uint32, numFiles)
	for i, f := range files {
		namePosMap[i] = uint32(len(namesBlob))
		namesBlob = append(namesBlob, []byte(f.name)...)
		namesBlob = append(namesBlob, 0)
	}

	// Build data blob
	var dataBlob []byte
	dataPosMap := make([]uint32, numFiles)
	dataSizeMap := make([]uint32, numFiles)
	for i, f := range files {
		dataPosMap[i] = uint32(len(dataBlob))
		dataSizeMap[i] = uint32(len(f.data))
		dataBlob = append(dataBlob, []byte(f.data)...)
	}

	// Section offsets
	startOfIndex := uint32(headerSize)
	startOfEntries := startOfIndex + indexSize
	startOfNames := startOfEntries + numFiles*entrySize
	startOfData := startOfNames + uint32(len(namesBlob))

	le := binary.LittleEndian
	var buf bytes.Buffer
	w32 := func(v uint32) {
		b := [4]byte{}
		le.PutUint32(b[:], v)
		buf.Write(b[:])
	}

	buf.Write(magic[:])
	w32(1) // version
	w32(startOfIndex)
	w32(startOfEntries)
	w32(startOfNames)
	w32(startOfData)
	w32(indexSize)
	w32(numFiles * entrySize)
	w32(uint32(len(namesBlob)))
	w32(uint32(len(dataBlob)))

	buf.Write(make([]byte, indexSize)) // index blob

	for i := uint32(0); i < numFiles; i++ {
		w32(dataPosMap[i])
		w32(namePosMap[i])
		w32(dataSizeMap[i])
		w32(dataSizeMap[i]) // DataSize2
		w32(0)              // compressed
	}

	buf.Write(namesBlob)
	buf.Write(dataBlob)
	return buf.Bytes()
}

func writeTempPAK(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.pak")
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatalf("write temp PAK: %v", err)
	}
	return p
}

var steamMagic = [4]byte{'L', 'P', 'A', 'K'}
var gogMagic = [4]byte{'K', 'A', 'P', 'L'}

var testFiles = []struct{ name, data string }{
	{"classic/en/monkey1.000", "hello_000"},
	{"classic/en/monkey1.001", "hello_001_longer"},
	{"other/asset.dat", "asset-data"},
}

// PAK-001: Round-trip is idempotent for Steam magic.
func TestRoundTripSteam(t *testing.T) { testRoundTrip(t, steamMagic) }

// PAK-002: Round-trip preserves GOG magic (not rewritten to LPAK).
func TestRoundTripGOG(t *testing.T) { testRoundTrip(t, gogMagic) }

func testRoundTrip(t *testing.T, magic [4]byte) {
	t.Helper()

	raw := buildSyntheticPAK(t, magic, testFiles)
	inPath := writeTempPAK(t, raw)
	out1 := filepath.Join(t.TempDir(), "out1.pak")
	out2 := filepath.Join(t.TempDir(), "out2.pak")

	hdr, idxBlob, namesBlob, entries, err := pak.Read(inPath)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != len(testFiles) {
		t.Fatalf("entry count: got %d, want %d", len(entries), len(testFiles))
	}
	for i, e := range entries {
		if e.Name != testFiles[i].name {
			t.Errorf("entry[%d].Name = %q, want %q", i, e.Name, testFiles[i].name)
		}
		if string(e.Data) != testFiles[i].data {
			t.Errorf("entry[%d].Data = %q, want %q", i, e.Data, testFiles[i].data)
		}
	}

	if err := pak.Write(out1, hdr, idxBlob, namesBlob, entries); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Second round-trip must produce byte-identical output.
	hdr2, idxBlob2, namesBlob2, entries2, err := pak.Read(out1)
	if err != nil {
		t.Fatalf("Read (2nd): %v", err)
	}
	if err := pak.Write(out2, hdr2, idxBlob2, namesBlob2, entries2); err != nil {
		t.Fatalf("Write (2nd): %v", err)
	}

	data1, _ := os.ReadFile(out1)
	data2, _ := os.ReadFile(out2)
	if !bytes.Equal(data1, data2) {
		t.Error("second round-trip is not byte-identical to first")
	}

	// Magic must be preserved.
	if data1[0] != magic[0] || data1[1] != magic[1] || data1[2] != magic[2] || data1[3] != magic[3] {
		t.Errorf("magic not preserved: got %q, want %q", data1[:4], magic[:])
	}
}

// PAK-003: DataPos is recalculated correctly when a file grows.
func TestDataPosRecalculation(t *testing.T) {
	raw := buildSyntheticPAK(t, steamMagic, testFiles)
	inPath := writeTempPAK(t, raw)
	outPath := filepath.Join(t.TempDir(), "out.pak")

	hdr, idxBlob, namesBlob, entries, err := pak.Read(inPath)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	newData := []byte("this is a much much much bigger replacement for entry 1")
	entries[1].Data = newData

	if err := pak.Write(outPath, hdr, idxBlob, namesBlob, entries); err != nil {
		t.Fatalf("Write: %v", err)
	}

	_, _, _, outEntries, err := pak.Read(outPath)
	if err != nil {
		t.Fatalf("Read (output): %v", err)
	}
	if !bytes.Equal(outEntries[0].Data, []byte(testFiles[0].data)) {
		t.Error("entry[0] changed unexpectedly")
	}
	if !bytes.Equal(outEntries[1].Data, newData) {
		t.Errorf("entry[1].Data = %q, want %q", outEntries[1].Data, newData)
	}
	if !bytes.Equal(outEntries[2].Data, []byte(testFiles[2].data)) {
		t.Error("entry[2] changed unexpectedly")
	}
}

// PAK-004: Wrong magic → clear error.
func TestInvalidMagic(t *testing.T) {
	raw := buildSyntheticPAK(t, [4]byte{'X', 'P', 'A', 'K'}, testFiles)
	path := writeTempPAK(t, raw)
	_, _, _, _, err := pak.Read(path)
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
}

// PAK-005: File too small → error.
func TestFileTooSmall(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tiny.pak")
	os.WriteFile(p, []byte("LPAK"), 0644)
	_, _, _, _, err := pak.Read(p)
	if err == nil {
		t.Fatal("expected error for too-small file")
	}
}

// PAK-006: Classic file entries can be found by name after read.
func TestClassicFilesFound(t *testing.T) {
	files := []struct{ name, data string }{
		{"classic/en/monkey1.000", "data_000"},
		{"classic/en/monkey1.001", "data_001"},
		{"other/asset.dat", "x"},
	}
	raw := buildSyntheticPAK(t, gogMagic, files)
	path := writeTempPAK(t, raw)

	_, _, _, entries, err := pak.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	found := map[string]bool{}
	for _, e := range entries {
		found[e.Name] = true
	}
	if !found["classic/en/monkey1.000"] {
		t.Error("classic/en/monkey1.000 not found")
	}
	if !found["classic/en/monkey1.001"] {
		t.Error("classic/en/monkey1.001 not found")
	}
}
