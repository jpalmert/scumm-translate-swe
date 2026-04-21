#!/usr/bin/env python3
"""
Extract English hint strings from a Monkey Island SE PAK archive.

Reads Monkey1.pak, locates the hints/monkey1.hints.csv entry, and extracts
only the English hint strings using the index matrix structure.

The index matrix at offset 0x76B0 contains entries that cycle through
5 languages (EN, FR, DE, IT, ES). Each entry has up to 4 hint levels.
English entries are at indices where index % 5 == 0.

Output: tab-separated ADDR<TAB>MAX_LEN<TAB>English text, one per line.

Usage:
  python3 extract_hints.py Monkey1.pak output.txt
"""

import os
import sys
import struct
from pathlib import Path

INDEX_MATRIX_OFFSET = 0x76B0
NUM_LANGS = 5
MAX_HINTS_PER_SERIES = 4
MATRIX_ENTRY_SIZE = MAX_HINTS_PER_SERIES * 4  # 16 bytes
POOL_ALIGN = 16


def read_pak_entry(pak_path, entry_name):
    """Read a named entry from a PAK archive, return its data."""
    with open(pak_path, 'rb') as f:
        magic = f.read(4)
        if magic not in (b'LPAK', b'KAPL'):
            raise ValueError(f"Not a PAK file (magic={magic!r})")

        f.read(4)  # version
        f.read(4)  # startOfIndex
        start_of_entries = struct.unpack('<I', f.read(4))[0]
        start_of_names = struct.unpack('<I', f.read(4))[0]
        start_of_data = struct.unpack('<I', f.read(4))[0]
        f.read(4)  # sizeOfIndex
        size_of_entries = struct.unpack('<I', f.read(4))[0]
        size_of_names = struct.unpack('<I', f.read(4))[0]

        num_files = size_of_entries // 20

        f.seek(start_of_names)
        names_blob = f.read(size_of_names)

        f.seek(start_of_entries)
        for _ in range(num_files):
            data_pos = struct.unpack('<I', f.read(4))[0]
            name_pos = struct.unpack('<I', f.read(4))[0]
            data_size = struct.unpack('<I', f.read(4))[0]
            f.read(4)  # data_size2
            f.read(4)  # compressed

            end = names_blob.index(b'\x00', name_pos)
            name = names_blob[name_pos:end].decode('ascii', errors='replace')

            if name.lower() == entry_name.lower():
                f.seek(start_of_data + data_pos)
                return f.read(data_size)

    return None


def read_string_at(data, addr):
    """Read a null-terminated Latin-1 string and its 16-byte-aligned slot size."""
    end = addr
    while end < len(data) and data[end] != 0:
        end += 1
    text = data[addr:end].decode('latin-1')
    # Slot size: next 16-byte boundary after the null terminator
    nxt = end + 1
    if nxt % POOL_ALIGN != 0:
        nxt += POOL_ALIGN - (nxt % POOL_ALIGN)
    if nxt > len(data):
        nxt = len(data)
    slot_size = max(nxt - addr, len(text) + 1)
    return text, slot_size


def extract_english(data):
    """Extract all English hint strings using the index matrix."""
    if len(data) < INDEX_MATRIX_OFFSET + 4:
        raise ValueError("Hints data too short")

    first_u32 = struct.unpack('<I', data[INDEX_MATRIX_OFFSET:INDEX_MATRIX_OFFSET + 4])[0]
    num_entries = first_u32 // MATRIX_ENTRY_SIZE

    if num_entries % NUM_LANGS != 0:
        raise ValueError(f"Entry count {num_entries} not divisible by {NUM_LANGS}")

    english = []
    for i in range(0, num_entries, NUM_LANGS):  # every 5th entry = English
        for level in range(MAX_HINTS_PER_SERIES):
            base = INDEX_MATRIX_OFFSET + i * MATRIX_ENTRY_SIZE
            field_addr = base + level * 4
            rel_off = struct.unpack('<I', data[field_addr:field_addr + 4])[0]
            if rel_off == 0:
                continue
            abs_addr = rel_off + field_addr
            if abs_addr >= len(data):
                continue
            text, _ = read_string_at(data, abs_addr)
            if text:
                english.append((abs_addr, text))

    return english


def extract(pak_path, output_path):
    hints_data = read_pak_entry(pak_path, 'hints/monkey1.hints.csv')
    if hints_data is None:
        print("ERROR: hints/monkey1.hints.csv not found in PAK", file=sys.stderr)
        sys.exit(1)

    english = extract_english(hints_data)

    with open(output_path, 'w', encoding='utf-8') as out:
        out.write(f"# SE Hint Text — {len(english)} English strings\n")
        out.write("# Format: ADDR<TAB>English text\n")
        out.write("# ADDR is the absolute byte offset in the hints file (stable identifier)\n")
        out.write("#\n")

        for addr, text in english:
            out.write(f"{addr}\t{text}\n")

    print(f"  {len(english)} English strings -> {output_path}")


def main():
    if len(sys.argv) != 3:
        print(__doc__.strip())
        sys.exit(1)
    extract(sys.argv[1], sys.argv[2])


if __name__ == '__main__':
    main()
