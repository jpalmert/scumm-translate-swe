package charset

import (
	"encoding/binary"
	"testing"
)

// buildFakeMonkey1001 creates a minimal XOR-encoded MONKEY1.001 containing a
// LECF block → LFLF block → CHAR blocks with the given sizes. Returns the
// encoded bytes and the decoded (in-memory) bytes for inspection.
func buildFakeMonkey1001(charSizes []int) (encoded []byte) {
	// Build decoded form first, then XOR-encode.
	charTotal := 0
	for _, s := range charSizes {
		charTotal += s
	}

	lflf := 8 + charTotal // LFLF header + char data
	lecf := 8 + lflf      // LECF header + lflf

	buf := make([]byte, lecf)
	copy(buf[0:], "LECF")
	binary.BigEndian.PutUint32(buf[4:], uint32(lecf))

	copy(buf[8:], "LFLF")
	binary.BigEndian.PutUint32(buf[12:], uint32(lflf))

	pos := 16
	for i, s := range charSizes {
		copy(buf[pos:], "CHAR")
		binary.BigEndian.PutUint32(buf[pos+4:], uint32(s))
		// Fill body with ordinal byte so we can identify each block.
		for j := 8; j < s; j++ {
			buf[pos+j] = byte(i + 1)
		}
		pos += s
	}

	// XOR-encode.
	enc := make([]byte, len(buf))
	for i, b := range buf {
		enc[i] = b ^ xorKey
	}
	return enc
}

// decodeFake XOR-decodes and returns the raw decoded bytes.
func decodeFake(enc []byte) []byte {
	out := make([]byte, len(enc))
	for i, b := range enc {
		out[i] = b ^ xorKey
	}
	return out
}

func TestXorDecodeSymmetric(t *testing.T) {
	data := []byte{0x00, 0xFF, 0x42, xorKey}
	encoded := xorDecode(data)
	decoded := xorDecode(encoded)
	for i := range data {
		if decoded[i] != data[i] {
			t.Fatalf("xorDecode not symmetric at index %d: got %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestFindCharBlock(t *testing.T) {
	enc := buildFakeMonkey1001([]int{20, 30, 40})
	data := decodeFake(enc)

	for n, want := range []int{16, 36, 66} { // expected offsets (0-based, 1-indexed n)
		got, err := findCharBlock(data, n+1)
		if err != nil {
			t.Fatalf("findCharBlock(%d): %v", n+1, err)
		}
		if got != want {
			t.Errorf("findCharBlock(%d): got %d, want %d", n+1, got, want)
		}
	}
}

func TestFindCharBlockNotFound(t *testing.T) {
	enc := buildFakeMonkey1001([]int{20})
	data := decodeFake(enc)
	_, err := findCharBlock(data, 2)
	if err == nil {
		t.Fatal("expected error for missing second CHAR block")
	}
}

func TestFindContainingLFLF(t *testing.T) {
	enc := buildFakeMonkey1001([]int{20, 30})
	data := decodeFake(enc)

	charOffset := 16 // first CHAR block
	lflf, err := findContainingLFLF(data, charOffset)
	if err != nil {
		t.Fatalf("findContainingLFLF: %v", err)
	}
	if lflf != 8 {
		t.Errorf("expected LFLF at offset 8, got %d", lflf)
	}
}

func TestPatchCharBlockSizeMismatch(t *testing.T) {
	// Build file with CHAR_0001 of size 100; patch expects originalChar0001Size (2609).
	enc := buildFakeMonkey1001([]int{100, 200, 300})
	_, err := patchCharBlock(decodeFake(enc), "CHAR_0001", []byte("replacement"), originalChar0001Size)
	if err == nil {
		t.Fatal("expected error for size mismatch")
	}
}

func TestPatchCharBlockRoundtrip(t *testing.T) {
	orig := originalChar0001Size
	// Build fake data with correct original size.
	enc := buildFakeMonkey1001([]int{orig, 50})

	newBlock := make([]byte, orig+28)
	copy(newBlock, "CHAR")
	binary.BigEndian.PutUint32(newBlock[4:], uint32(len(newBlock)))
	for i := 8; i < len(newBlock); i++ {
		newBlock[i] = 0xAB
	}

	patched, err := patchCharBlock(decodeFake(enc), "CHAR_0001", newBlock, orig)
	if err != nil {
		t.Fatalf("patchCharBlock: %v", err)
	}

	// Verify the new block is at offset 16.
	if string(patched[16:20]) != "CHAR" {
		t.Errorf("expected CHAR tag at 16, got %q", patched[16:20])
	}
	newSize := int(binary.BigEndian.Uint32(patched[20:]))
	if newSize != len(newBlock) {
		t.Errorf("new block size: got %d, want %d", newSize, len(newBlock))
	}

	// Verify LFLF size updated.
	lflf := int(binary.BigEndian.Uint32(patched[12:]))
	wantLFLF := 8 + orig + 50 + 28
	if lflf != wantLFLF {
		t.Errorf("LFLF size: got %d, want %d", lflf, wantLFLF)
	}

	// Verify LECF size updated.
	lecf := int(binary.BigEndian.Uint32(patched[4:]))
	wantLECF := 8 + wantLFLF
	if lecf != wantLECF {
		t.Errorf("LECF size: got %d, want %d", lecf, wantLECF)
	}

	// Verify the block after CHAR_0001 (the 50-byte one) is intact.
	afterOffset := 16 + len(newBlock)
	if string(patched[afterOffset:afterOffset+4]) != "CHAR" {
		t.Errorf("second CHAR block not at expected position")
	}
	secondSize := int(binary.BigEndian.Uint32(patched[afterOffset+4:]))
	if secondSize != 50 {
		t.Errorf("second CHAR block size: got %d, want 50", secondSize)
	}
}

func TestAddToSize(t *testing.T) {
	data := make([]byte, 12)
	binary.BigEndian.PutUint32(data[4:], 100)
	addToSize(data, 0, 28)
	got := int(binary.BigEndian.Uint32(data[4:]))
	if got != 128 {
		t.Errorf("addToSize: got %d, want 128", got)
	}
}

// buildDCHRBody builds a raw (plain) DCHR block with the given charset offsets.
// Format: 2-byte count, count disk bytes, count×4-byte LE offsets.
func buildDCHRBody(offsets []uint32) []byte {
	count := len(offsets)
	bodySize := 2 + count + count*4 // count(2) + disk bytes(count) + offsets(count*4)
	blockSize := 8 + bodySize

	buf := make([]byte, blockSize)
	copy(buf[0:], "DCHR")
	binary.BigEndian.PutUint32(buf[4:], uint32(blockSize))
	binary.LittleEndian.PutUint16(buf[8:], uint16(count))
	// Disk bytes at buf[10 : 10+count]
	for i := range offsets {
		buf[10+i] = 1
	}
	// Offsets at buf[10+count : 10+count+count*4]
	offsetsBase := 10 + count
	for i, off := range offsets {
		binary.LittleEndian.PutUint32(buf[offsetsBase+i*4:], off)
	}
	return buf
}

// buildFakeMonkey1000 creates a minimal XOR-encoded MONKEY1.000 with a DCHR
// block containing the given charset offsets (LFLF-body-relative, LE32).
func buildFakeMonkey1000(offsets []uint32) []byte {
	buf := buildDCHRBody(offsets)
	enc := make([]byte, len(buf))
	for i, b := range buf {
		enc[i] = b ^ xorKey
	}
	return enc
}

// buildFakePlainMonkey1000 creates an unencoded MONKEY1.000 (as found in SE PAK).
func buildFakePlainMonkey1000(offsets []uint32) []byte {
	return buildDCHRBody(offsets)
}

// readOffsets decodes a XOR-encoded fake MONKEY1.000 and reads back charset offsets.
func readOffsets(enc []byte) []uint32 {
	data := make([]byte, len(enc))
	for i, b := range enc {
		data[i] = b ^ xorKey
	}
	return readPlainOffsets(data)
}

// readPlainOffsets reads charset offsets from a plain (non-encoded) DCHR block.
func readPlainOffsets(data []byte) []uint32 {
	count := int(binary.LittleEndian.Uint16(data[8:]))
	offsetsBase := 10 + count
	offsets := make([]uint32, count)
	for i := range offsets {
		offsets[i] = binary.LittleEndian.Uint32(data[offsetsBase+i*4:])
	}
	return offsets
}

func TestPatchMonkey1000NoDCHR(t *testing.T) {
	// Neither XOR-encoded nor plain data contains "DCHR".
	data := make([]byte, 16) // all zeros — no DCHR in either form
	_, err := PatchMonkey1000(data)
	if err == nil {
		t.Fatal("expected error when DCHR block is missing")
	}
}

func TestPatchMonkey1000OffsetsShifted(t *testing.T) {
	// Provide offsets matching the known layout:
	//   CHAR_0001: 98401  → no change (first modified block)
	//   CHAR_0002: 101010 → +16 (char0001Delta)
	//   CHAR_0003: 105618 → +16 (char0001Delta)
	//   CHAR_0004: 107689 → +82 (char0001Delta+char0003Delta = 16+66)
	//   CHAR_0006: 112479 → +82
	input := []uint32{98401, 101010, 105618, 107689, 112479}
	want := []uint32{98401, 101026, 105634, 107771, 112561}

	// Test with XOR-encoded input.
	enc := buildFakeMonkey1000(input)
	patched, err := PatchMonkey1000(enc)
	if err != nil {
		t.Fatalf("PatchMonkey1000 (encoded): %v", err)
	}
	got := readOffsets(patched)
	if len(got) != len(want) {
		t.Fatalf("encoded: got %d offsets, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("encoded: offset[%d]: got %d, want %d", i, got[i], w)
		}
	}

	// Test with plain (non-encoded) input — as found in the SE PAK.
	plain := buildFakePlainMonkey1000(input)
	patchedPlain, err := PatchMonkey1000(plain)
	if err != nil {
		t.Fatalf("PatchMonkey1000 (plain): %v", err)
	}
	gotPlain := readPlainOffsets(patchedPlain)
	if len(gotPlain) != len(want) {
		t.Fatalf("plain: got %d offsets, want %d", len(gotPlain), len(want))
	}
	for i, w := range want {
		if gotPlain[i] != w {
			t.Errorf("plain: offset[%d]: got %d, want %d", i, gotPlain[i], w)
		}
	}
}

func TestPatchMonkey1001PlainInput(t *testing.T) {
	// Verify PatchMonkey1001 handles plain (non-XOR-encoded) input — as found
	// in the SE PAK. Build a minimal LECF/LFLF/CHAR structure without encoding.
	orig1 := originalChar0001Size
	orig3 := originalChar0003Size
	enc := buildFakeMonkey1001([]int{orig1, 50, orig3})
	plain := decodeFake(enc) // XOR-decode → plain LECF data

	// Sanity: plain data starts with LECF.
	if string(plain[0:4]) != "LECF" {
		t.Fatal("test setup: plain data should start with LECF")
	}

	patched, err := PatchMonkey1001(plain)
	if err != nil {
		t.Fatalf("PatchMonkey1001 (plain): %v", err)
	}

	// Output should also be plain (not XOR-encoded).
	if string(patched[0:4]) != "LECF" {
		t.Errorf("output should start with LECF, got %q", patched[0:4])
	}

	// CHAR_0001 and CHAR_0003 should have grown.
	if len(patched) <= len(plain) {
		t.Errorf("patched size %d should be larger than plain size %d", len(patched), len(plain))
	}
}

func TestPatchMonkey1001EncodedInput(t *testing.T) {
	// Verify PatchMonkey1001 handles XOR-encoded input — as found on disk in
	// the classic game.
	orig1 := originalChar0001Size
	orig3 := originalChar0003Size
	enc := buildFakeMonkey1001([]int{orig1, 50, orig3})

	// Sanity: encoded data does NOT start with LECF.
	if string(enc[0:4]) == "LECF" {
		t.Fatal("test setup: encoded data should not start with LECF")
	}

	patched, err := PatchMonkey1001(enc)
	if err != nil {
		t.Fatalf("PatchMonkey1001 (encoded): %v", err)
	}

	// Output should also be XOR-encoded (not plain LECF).
	if string(patched[0:4]) == "LECF" {
		t.Errorf("output should be XOR-encoded, not plain LECF")
	}

	// After decoding, should start with LECF and be larger.
	decoded := decodeFake(patched)
	if string(decoded[0:4]) != "LECF" {
		t.Errorf("decoded output should start with LECF, got %q", decoded[0:4])
	}
	if len(patched) <= len(enc) {
		t.Errorf("patched size %d should be larger than encoded size %d", len(patched), len(enc))
	}
}

// buildFakeWithLOFF builds a minimal decoded LECF with a LOFF block containing
// the given room LFLF-body offsets (file-absolute, i.e. LFLF_start+8).
func buildFakeWithLOFF(lflfBodyOffsets []uint32) []byte {
	count := len(lflfBodyOffsets)
	loffBodySize := 1 + count*5
	loffSize := 8 + loffBodySize
	total := 8 + loffSize

	buf := make([]byte, total)
	copy(buf[0:], "LECF")
	binary.BigEndian.PutUint32(buf[4:], uint32(total))
	copy(buf[8:], "LOFF")
	binary.BigEndian.PutUint32(buf[12:], uint32(loffSize))
	buf[16] = byte(count)
	for i, off := range lflfBodyOffsets {
		base := 17 + i*5
		buf[base] = byte(i + 1) // room_id
		binary.LittleEndian.PutUint32(buf[base+1:], off)
	}
	return buf
}

func TestUpdateLOFF(t *testing.T) {
	// 3 rooms: room 1 body at 500 (before charset), room 2 body at 1008
	// (charset LFLF starts at 1000, body at 1000+8=1008), room 3 body at 2008.
	data := buildFakeWithLOFF([]uint32{500, 1008, 2008})

	if err := updateLOFF(data, 1000, 28); err != nil {
		t.Fatalf("updateLOFF: %v", err)
	}

	readOffset := func(entryIdx int) uint32 {
		base := 17 + entryIdx*5
		return binary.LittleEndian.Uint32(data[base+1:])
	}

	if got := readOffset(0); got != 500 {
		t.Errorf("entry[0]: got %d, want 500 (before charset LFLF — unchanged)", got)
	}
	if got := readOffset(1); got != 1008 {
		t.Errorf("entry[1]: got %d, want 1008 (charset LFLF itself — unchanged)", got)
	}
	if got := readOffset(2); got != 2036 {
		t.Errorf("entry[2]: got %d, want 2036 (after charset LFLF — shifted by 28)", got)
	}
}

func TestUpdateLOFFNoBlock(t *testing.T) {
	// Test data has no LOFF block — updateLOFF must be a no-op.
	enc := buildFakeMonkey1001([]int{20, 30})
	data := decodeFake(enc)
	original := append([]byte(nil), data...)

	if err := updateLOFF(data, 8, 10); err != nil {
		t.Fatalf("updateLOFF with no LOFF: %v", err)
	}
	for i := range data {
		if data[i] != original[i] {
			t.Errorf("data[%d] changed when no LOFF present", i)
		}
	}
}

func TestPatchMonkey1000EarlyOffsetsUnchanged(t *testing.T) {
	// Offsets before CHAR_0001 must not be modified.
	input := []uint32{1000, 50000, 98401}
	enc := buildFakeMonkey1000(input)

	patched, err := PatchMonkey1000(enc)
	if err != nil {
		t.Fatalf("PatchMonkey1000: %v", err)
	}

	got := readOffsets(patched)
	for i, w := range []uint32{1000, 50000, 98401} {
		if got[i] != w {
			t.Errorf("offset[%d]: got %d, want %d (should be unchanged)", i, got[i], w)
		}
	}
}
