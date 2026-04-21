#!/usr/bin/env python3
"""
Extract English UI text from a Monkey Island SE uiText.info file.

Format: 511 entries, each = 6 x 256 bytes (key, EN, FR, IT, DE, ES).
Each 256-byte slot is null-terminated and space-padded.

Output: tab-separated KEY<TAB>EN_TEXT, one per line.

Usage:
  python3 extract_uitext.py localization/uiText.info output.txt
"""

import sys
from pathlib import Path

FIELD_SIZE = 256
FIELD_COUNT = 6  # key + 5 languages
ENTRY_SIZE = FIELD_COUNT * FIELD_SIZE


def decode_field(slot: bytes) -> str:
    """Decode a 256-byte null-terminated Latin-1 field."""
    end = slot.find(b'\x00')
    if end == -1:
        end = len(slot)
    return slot[:end].decode('latin-1')


# Only extract entries with these key prefixes — these are SE-specific UI
# strings that don't exist in classic mode. Object names, verbs, and misc
# game strings are handled by scummtr injection and don't need separate
# translation here.
SE_KEY_PREFIXES = ('MENU_', 'OVERLAY_', 'CREDITS_', 'LOADING')


def extract(info_path: str, output_path: str) -> None:
    data = Path(info_path).read_bytes()
    if len(data) % ENTRY_SIZE != 0:
        print(f"ERROR: file size {len(data)} is not a multiple of {ENTRY_SIZE}", file=sys.stderr)
        sys.exit(1)

    num_entries = len(data) // ENTRY_SIZE
    written = 0

    with open(output_path, 'w', encoding='utf-8') as out:
        out.write(f"# SE UI Text — SE-specific entries from {Path(info_path).name}\n")
        out.write("# Format: KEY<TAB>English text\n")
        out.write("#\n")

        for i in range(num_entries):
            base = i * ENTRY_SIZE
            key = decode_field(data[base:base + FIELD_SIZE])
            if not key.startswith(SE_KEY_PREFIXES):
                continue
            en = decode_field(data[base + FIELD_SIZE:base + 2 * FIELD_SIZE])
            out.write(f"{key}\t{en}\n")
            written += 1

    print(f"  {written} SE entries (of {num_entries} total) -> {output_path}")


def main():
    if len(sys.argv) != 3:
        print(__doc__.strip())
        sys.exit(1)
    extract(sys.argv[1], sys.argv[2])


if __name__ == '__main__':
    main()
