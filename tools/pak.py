#!/usr/bin/env python3
"""
PAK archive extractor and repacker for Monkey Island Special Edition.
Supports MI:SE (game 1) and MI2:SE (game 2).

PAK format (little-endian):
  Header (40 bytes = 0x28):
    +0x00  4B  magic "LPAK"
    +0x04  4B  version
    +0x08  4B  startOfIndex
    +0x0C  4B  startOfFileEntries
    +0x10  4B  startOfFileNames
    +0x14  4B  startOfData
    +0x18  4B  sizeOfIndex
    +0x1C  4B  sizeOfFileEntries
    +0x20  4B  sizeOfFileNames
    +0x24  4B  sizeOfData

  File Entry (20 bytes = 0x14):
    +0x00  4B  fileDataPos   (relative to startOfData)
    +0x04  4B  fileNamePos   (relative to startOfFileNames)
    +0x08  4B  dataSize
    +0x0C  4B  dataSize2     (always == dataSize)
    +0x10  4B  compressed    (always 0)

  MI:SE  - files stored in entry-list order; fileNamePos is valid
  MI2:SE - files stored sorted by fileDataPos; fileNamePos is broken (read names sequentially)

Usage:
  python3 pak.py extract Monkey1.pak output_dir/
  python3 pak.py repack  output_dir/ Monkey1_modified.pak Monkey1.pak
"""

import os
import sys
import struct
from pathlib import Path

MAGIC = b'LPAK'
MAGIC_GOG = b'KAPL'  # GOG version uses reversed magic bytes
HEADER_SIZE = 0x28
ENTRY_SIZE = 20


def _read_u32(f):
    return struct.unpack('<I', f.read(4))[0]


def _write_u32(f, val):
    f.write(struct.pack('<I', val))


def _read_cstring(data, offset):
    end = data.index(b'\x00', offset)
    return data[offset:end].decode('ascii', errors='replace')


def detect_game(pak_path):
    """Return 1 for MI:SE, 2 for MI2:SE based on filename heuristic."""
    name = Path(pak_path).name.lower()
    if 'monkey2' in name or 'mi2' in name:
        return 2
    return 1


def extract(pak_path, output_dir, game=None):
    """Extract all files from a PAK archive into output_dir."""
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    if game is None:
        game = detect_game(pak_path)

    with open(pak_path, 'rb') as f:
        magic = f.read(4)
        if magic not in (MAGIC, MAGIC_GOG):
            raise ValueError(f"Not a PAK file (magic={magic!r})")

        version          = _read_u32(f)
        start_of_index   = _read_u32(f)
        start_of_entries = _read_u32(f)
        start_of_names   = _read_u32(f)
        start_of_data    = _read_u32(f)
        size_of_index    = _read_u32(f)
        size_of_entries  = _read_u32(f)
        size_of_names    = _read_u32(f)
        size_of_data     = _read_u32(f)

        num_files = size_of_entries // ENTRY_SIZE

        # Read file entries
        f.seek(start_of_entries)
        entries = []
        for _ in range(num_files):
            data_pos   = _read_u32(f)
            name_pos   = _read_u32(f)
            data_size  = _read_u32(f)
            data_size2 = _read_u32(f)
            compressed = _read_u32(f)
            entries.append({
                'data_pos': data_pos,
                'name_pos': name_pos,
                'data_size': data_size,
                'compressed': compressed,
            })

        # Read filenames section
        f.seek(start_of_names)
        names_blob = f.read(size_of_names)

        # Resolve filenames
        if game == 2:
            # MI2:SE: fileNamePos is broken; read sequentially
            offset = 0
            for entry in entries:
                name = _read_cstring(names_blob, offset)
                entry['name'] = name
                offset += len(name) + 1
        else:
            for entry in entries:
                entry['name'] = _read_cstring(names_blob, entry['name_pos'])

        # Determine extraction order (MI2:SE stores data sorted by fileDataPos)
        if game == 2:
            extraction_order = sorted(range(num_files), key=lambda i: entries[i]['data_pos'])
        else:
            extraction_order = list(range(num_files))

        # Extract files
        for i in extraction_order:
            entry = entries[i]
            abs_pos = start_of_data + entry['data_pos']
            f.seek(abs_pos)
            data = f.read(entry['data_size'])

            out_path = output_dir / entry['name']
            out_path.parent.mkdir(parents=True, exist_ok=True)
            out_path.write_bytes(data)
            print(f"  extracted: {entry['name']} ({entry['data_size']} bytes)")

    print(f"\nExtracted {num_files} files to {output_dir}")


def repack(input_dir, output_pak, reference_pak, game=None):
    """
    Repack a directory of files back into a PAK archive.
    Uses reference_pak for header structure (entry order, filenames, index).
    Modified files are picked up from input_dir; unchanged files are taken
    from the reference PAK data.
    """
    input_dir = Path(input_dir)

    if game is None:
        game = detect_game(reference_pak)

    with open(reference_pak, 'rb') as f:
        magic = f.read(4)
        if magic not in (MAGIC, MAGIC_GOG):
            raise ValueError(f"Not a PAK file (magic={magic!r})")

        version          = _read_u32(f)
        start_of_index   = _read_u32(f)
        start_of_entries = _read_u32(f)
        start_of_names   = _read_u32(f)
        start_of_data    = _read_u32(f)
        size_of_index    = _read_u32(f)
        size_of_entries  = _read_u32(f)
        size_of_names    = _read_u32(f)
        size_of_data     = _read_u32(f)

        num_files = size_of_entries // ENTRY_SIZE

        f.seek(start_of_index)
        index_blob = f.read(size_of_index)

        f.seek(start_of_entries)
        entries = []
        for _ in range(num_files):
            data_pos   = _read_u32(f)
            name_pos   = _read_u32(f)
            data_size  = _read_u32(f)
            data_size2 = _read_u32(f)
            compressed = _read_u32(f)
            entries.append({
                'data_pos': data_pos,
                'name_pos': name_pos,
                'data_size': data_size,
                'data_size2': data_size2,
                'compressed': compressed,
            })

        f.seek(start_of_names)
        names_blob = f.read(size_of_names)

        # Resolve filenames
        if game == 2:
            offset = 0
            for entry in entries:
                name = _read_cstring(names_blob, offset)
                entry['name'] = name
                offset += len(name) + 1
        else:
            for entry in entries:
                entry['name'] = _read_cstring(names_blob, entry['name_pos'])

        # Load all original file data from reference
        for entry in entries:
            abs_pos = start_of_data + entry['data_pos']
            f.seek(abs_pos)
            entry['original_data'] = f.read(entry['data_size'])

    # Override with modified files from input_dir
    for entry in entries:
        candidate = input_dir / entry['name']
        if candidate.exists():
            entry['new_data'] = candidate.read_bytes()
            if len(entry['new_data']) != entry['data_size']:
                print(f"  size change: {entry['name']}  {entry['data_size']} -> {len(entry['new_data'])} bytes")
        else:
            entry['new_data'] = entry['original_data']

    # Determine write order
    if game == 2:
        write_order = sorted(range(num_files), key=lambda i: entries[i]['data_pos'])
    else:
        write_order = list(range(num_files))

    # Build updated entries: recalculate data_pos based on new sizes
    offset_map = {}  # original data_pos -> new data_pos
    current_pos = 0

    for i in write_order:
        entry = entries[i]
        offset_map[entry['data_pos']] = current_pos
        current_pos += len(entry['new_data'])

    new_size_of_data = current_pos

    # Update entries with new positions and sizes
    for entry in entries:
        entry['new_data_pos'] = offset_map[entry['data_pos']]
        entry['new_data_size'] = len(entry['new_data'])

    # Write output PAK
    with open(output_pak, 'wb') as out:
        # Header — preserve original magic (LPAK for Steam, KAPL for GOG)
        out.write(magic)
        _write_u32(out, version)
        _write_u32(out, start_of_index)
        _write_u32(out, start_of_entries)
        _write_u32(out, start_of_names)
        _write_u32(out, start_of_data)
        _write_u32(out, size_of_index)
        _write_u32(out, size_of_entries)
        _write_u32(out, size_of_names)
        _write_u32(out, new_size_of_data)

        # Index section (unchanged)
        out.seek(start_of_index)
        out.write(index_blob)

        # File entries (updated positions/sizes)
        out.seek(start_of_entries)
        for entry in entries:
            _write_u32(out, entry['new_data_pos'])
            _write_u32(out, entry['name_pos'])
            _write_u32(out, entry['new_data_size'])
            _write_u32(out, entry['new_data_size'])
            _write_u32(out, entry['compressed'])

        # Filenames section (unchanged)
        out.seek(start_of_names)
        out.write(names_blob)

        # Data section
        out.seek(start_of_data)
        for i in write_order:
            out.write(entries[i]['new_data'])

    print(f"\nRepacked {num_files} files to {output_pak}")


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    cmd = sys.argv[1]

    if cmd == 'extract' and len(sys.argv) >= 4:
        game = int(sys.argv[4]) if len(sys.argv) > 4 else None
        extract(sys.argv[2], sys.argv[3], game)
    elif cmd == 'repack' and len(sys.argv) >= 5:
        game = int(sys.argv[5]) if len(sys.argv) > 5 else None
        repack(sys.argv[2], sys.argv[3], sys.argv[4], game)
    else:
        print(__doc__)
        sys.exit(1)


if __name__ == '__main__':
    main()
