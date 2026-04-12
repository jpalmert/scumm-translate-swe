// Package pak reads and writes Monkey Island Special Edition PAK archive files.
//
// PAK format (little-endian, all offsets absolute):
//
//	Header (40 bytes):
//	  +0x00  4B  magic "LPAK" (Steam) or "KAPL" (GOG)
//	  +0x04  4B  version
//	  +0x08  4B  startOfIndex
//	  +0x0C  4B  startOfFileEntries
//	  +0x10  4B  startOfFileNames
//	  +0x14  4B  startOfData
//	  +0x18  4B  sizeOfIndex
//	  +0x1C  4B  sizeOfFileEntries
//	  +0x20  4B  sizeOfFileNames
//	  +0x24  4B  sizeOfData
//
//	File Entry (20 bytes):
//	  +0x00  4B  fileDataPos   (byte offset relative to startOfData)
//	  +0x04  4B  fileNamePos   (byte offset relative to startOfFileNames)
//	  +0x08  4B  dataSize
//	  +0x0C  4B  dataSize2     (always == dataSize)
//	  +0x10  4B  compressed    (always 0)
package pak

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

var validMagics = [][4]byte{
	{'L', 'P', 'A', 'K'}, // Steam
	{'K', 'A', 'P', 'L'}, // GOG
}

// Header mirrors the 40-byte PAK file header.
type Header struct {
	Magic          [4]byte
	Version        uint32
	StartOfIndex   uint32
	StartOfEntries uint32
	StartOfNames   uint32
	StartOfData    uint32
	SizeOfIndex    uint32
	SizeOfEntries  uint32
	SizeOfNames    uint32
	SizeOfData     uint32
}

// Entry represents one file stored inside the PAK.
type Entry struct {
	NamePos    uint32 // offset into names section (preserved verbatim on write)
	Compressed uint32 // always 0 in known PAK files
	Name       string
	Data       []byte
}

// Read reads a PAK file and returns its header, raw index and names blobs
// (copied verbatim on write), and all file entries with their data loaded.
func Read(path string) (*Header, []byte, []byte, []*Entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if len(raw) < 40 {
		return nil, nil, nil, nil, fmt.Errorf("file too small to be a PAK")
	}

	// Validate magic
	var magic [4]byte
	copy(magic[:], raw[0:4])
	valid := false
	for _, m := range validMagics {
		if magic == m {
			valid = true
			break
		}
	}
	if !valid {
		return nil, nil, nil, nil, fmt.Errorf("not a PAK file (magic=%q) — is this really Monkey1.pak?", magic)
	}

	le := binary.LittleEndian
	hdr := &Header{
		Magic:          magic,
		Version:        le.Uint32(raw[4:]),
		StartOfIndex:   le.Uint32(raw[8:]),
		StartOfEntries: le.Uint32(raw[12:]),
		StartOfNames:   le.Uint32(raw[16:]),
		StartOfData:    le.Uint32(raw[20:]),
		SizeOfIndex:    le.Uint32(raw[24:]),
		SizeOfEntries:  le.Uint32(raw[28:]),
		SizeOfNames:    le.Uint32(raw[32:]),
		SizeOfData:     le.Uint32(raw[36:]),
	}

	// Index blob — copied verbatim on write
	indexBlob := make([]byte, hdr.SizeOfIndex)
	copy(indexBlob, raw[hdr.StartOfIndex:])

	// Names blob — copied verbatim on write
	namesBlob := make([]byte, hdr.SizeOfNames)
	copy(namesBlob, raw[hdr.StartOfNames:])

	// Parse file entries (20 bytes each)
	numFiles := hdr.SizeOfEntries / 20
	entries := make([]*Entry, numFiles)
	for i := uint32(0); i < numFiles; i++ {
		off := hdr.StartOfEntries + i*20
		dataPos := le.Uint32(raw[off:])
		namePos := le.Uint32(raw[off+4:])
		dataSize := le.Uint32(raw[off+8:])
		compressed := le.Uint32(raw[off+16:])

		// Resolve null-terminated filename from the names blob
		ns := namePos
		ne := ns
		for ne < uint32(len(namesBlob)) && namesBlob[ne] != 0 {
			ne++
		}
		name := string(namesBlob[ns:ne])

		// Copy file data
		dataStart := hdr.StartOfData + dataPos
		data := make([]byte, dataSize)
		copy(data, raw[dataStart:dataStart+dataSize])

		entries[i] = &Entry{
			NamePos:    namePos,
			Compressed: compressed,
			Name:       name,
			Data:       data,
		}
	}

	return hdr, indexBlob, namesBlob, entries, nil
}

// Write writes entries back to a PAK file, recalculating DataPos values from
// scratch so size changes in individual entries are handled correctly.
// The index and names blobs are written verbatim (unchanged from the original).
// The original magic bytes (LPAK or KAPL) are preserved.
func Write(path string, hdr *Header, indexBlob, namesBlob []byte, entries []*Entry) error {
	var buf bytes.Buffer

	// Calculate new data positions (entries packed with no gaps)
	dataPositions := make([]uint32, len(entries))
	pos := uint32(0)
	for i, e := range entries {
		dataPositions[i] = pos
		pos += uint32(len(e.Data))
	}
	newSizeOfData := pos

	// --- Header ---
	buf.Write(hdr.Magic[:])
	writeU32(&buf, hdr.Version)
	writeU32(&buf, hdr.StartOfIndex)
	writeU32(&buf, hdr.StartOfEntries)
	writeU32(&buf, hdr.StartOfNames)
	writeU32(&buf, hdr.StartOfData)
	writeU32(&buf, hdr.SizeOfIndex)
	writeU32(&buf, hdr.SizeOfEntries)
	writeU32(&buf, hdr.SizeOfNames)
	writeU32(&buf, newSizeOfData)

	// --- Index section (unchanged) ---
	padTo(&buf, hdr.StartOfIndex)
	buf.Write(indexBlob)

	// --- File entries (updated DataPos and DataSize, everything else verbatim) ---
	padTo(&buf, hdr.StartOfEntries)
	for i, e := range entries {
		sz := uint32(len(e.Data))
		writeU32(&buf, dataPositions[i])
		writeU32(&buf, e.NamePos)
		writeU32(&buf, sz)
		writeU32(&buf, sz)
		writeU32(&buf, e.Compressed)
	}

	// --- Names section (unchanged) ---
	padTo(&buf, hdr.StartOfNames)
	buf.Write(namesBlob)

	// --- Data section ---
	padTo(&buf, hdr.StartOfData)
	for _, e := range entries {
		buf.Write(e.Data)
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

func writeU32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

// padTo writes zero bytes until buf.Len() == target.
func padTo(buf *bytes.Buffer, target uint32) {
	for uint32(buf.Len()) < target {
		buf.WriteByte(0)
	}
}
