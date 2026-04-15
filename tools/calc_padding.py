#!/usr/bin/env python3
"""
Calculate required @ padding for Swedish object names based on dynamic_names.json.

For each object that has runtime name replacements (setObjectName), this script:
1. Reads the Swedish translations from swedish.txt
2. Computes the SCUMM byte length of each translated name
3. Determines the required buffer size (max of OBNA name and all replacements)
4. Reports which lines need padding and by how much
5. Optionally applies the padding (--apply flag)

The SCUMM byte length accounts for escape codes: \\NNN = 1 byte, all other
characters = 1 byte each. Swedish UTF-8 characters (å, ä, ö, etc.) should
already be encoded as \\NNN escape codes in swedish.txt by the injection pipeline.

Usage:
    python3 tools/calc_padding.py [--apply] [--json mapping] [--translation file]

    --apply:       modify swedish.txt in place to add/fix padding
    --json:        path to dynamic_names.json (default: translation/monkey1/dynamic_names.json)
    --translation: path to swedish.txt (default: translation/monkey1/swedish.txt)
    --verbose:     show all objects, not just those needing changes
"""

import argparse
import json
import os
import re
import sys

# Swedish UTF-8 -> SCUMM escape code mappings (must match internal/classic/classic.go)
SWEDISH_CHAR_MAP = {
    'Å': '\\091', 'Ä': '\\092', 'Ö': '\\093',
    'å': '\\123', 'ä': '\\124', 'ö': '\\125',
    'é': '\\130', 'ê': '\\136', '®': '\\015',
}


def encode_swedish(text):
    """Encode UTF-8 Swedish chars to SCUMM escape codes and strip opcode prefix."""
    for char, esc in SWEDISH_CHAR_MAP.items():
        text = text.replace(char, esc)
    # Strip leading (XX) opcode prefix
    if text.startswith('(') and ')' in text:
        text = text[text.index(')') + 1:]
    return text


def scumm_byte_len(text):
    """Count SCUMM bytes. \\NNN = 1 byte, each other char = 1 byte."""
    n, i = 0, 0
    while i < len(text):
        if text[i] == '\\' and i + 3 < len(text) and text[i+1:i+4].isdigit():
            n += 1
            i += 4
        else:
            n += 1
            i += 1
    return n


def pad_to_length(text, target_scumm_len):
    """Pad text with @ to reach target SCUMM byte length."""
    current = scumm_byte_len(text)
    if current < target_scumm_len:
        text += '@' * (target_scumm_len - current)
    return text


def strip_trailing_at(text):
    """Remove trailing @ characters."""
    return text.rstrip('@')


def parse_swedish_lines(filepath):
    """Parse swedish.txt into list of (header, text, line_number)."""
    lines = []
    with open(filepath, 'r', encoding='utf-8') as f:
        for i, raw in enumerate(f, 1):
            raw = raw.rstrip('\r\n')
            if raw.startswith('[') and ']' in raw:
                idx = raw.index(']')
                lines.append((raw[:idx+1], raw[idx+1:], i))
    return lines


def find_lines_for_object(swedish_lines, obj_id, mapping):
    """Find the swedish.txt lines that correspond to an object's OBNA and replacement names.

    Returns dict with:
      obna_lines: [(header, text, line_num)] - the OBNA line(s) for this object
      replacement_lines: [(header, text, line_num, replacement_entry)] - the (54)/(D4) lines
    """
    obna_header_pattern = f"OBNA#{obj_id:04d}]"

    obna_lines = []
    replacement_lines = []

    for header, text, line_num in swedish_lines:
        if obna_header_pattern in header:
            obna_lines.append((header, text, line_num))

    # For replacement lines, we need to match by the scummtr header format
    # The replacement names appear as regular text lines in swedish.txt with
    # their own headers (VERB#, SCRP#, LSCR#, ENCD#, EXCD#)
    # We can't directly match them by object ID from the header alone —
    # the header shows the script/resource they're IN, not the target object.
    # This is handled by the mapping from dynamic_names.json.

    return {
        "obna_lines": obna_lines,
    }


def main():
    parser = argparse.ArgumentParser(description="Calculate @ padding for Swedish object names")
    parser.add_argument("--apply", action="store_true", help="Apply padding to swedish.txt")
    parser.add_argument("--json", default=None, help="Path to dynamic_names.json")
    parser.add_argument("--translation", default=None, help="Path to swedish.txt")
    parser.add_argument("--verbose", action="store_true", help="Show all objects")
    args = parser.parse_args()

    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    json_path = args.json or os.path.join(repo, "translation", "monkey1", "dynamic_names.json")
    sv_path = args.translation or os.path.join(repo, "translation", "monkey1", "swedish.txt")

    if not os.path.isfile(json_path):
        print(f"Error: {json_path} not found. Run tools/find_dynamic_names.py first.", file=sys.stderr)
        sys.exit(1)

    with open(json_path) as f:
        mapping = json.load(f)

    swedish_lines = parse_swedish_lines(sv_path)

    # Build a lookup: header -> (text, line_num, line_index)
    header_lookup = {}
    for idx, (header, text, line_num) in enumerate(swedish_lines):
        if header not in header_lookup:
            header_lookup[header] = []
        header_lookup[header].append((text, line_num, idx))

    print("=" * 70)
    print("PADDING ANALYSIS")
    print("=" * 70)
    print()

    needs_change = 0
    total_objects = 0
    problems = []

    for obj_id_str, data in sorted(mapping["objects"].items(), key=lambda x: int(x[0])):
        obj_id = int(obj_id_str)
        if not data["replacements"]:
            continue

        total_objects += 1
        obna = data.get("obna", "")
        obna_len = data.get("obna_len", 0)
        max_repl = data.get("max_replacement_len", 0)
        required_buffer = max(obna_len, max_repl)

        # Find the OBNA line in swedish.txt
        obna_header = f"[{obj_id // 256:03d}:OBNA#{obj_id:04d}]"
        # Actually, room number in header doesn't directly correspond to obj_id
        # Search all OBNA headers for this object number
        obna_matches = []
        for header, entries in header_lookup.items():
            if f"OBNA#{obj_id:04d}]" in header:
                for text, line_num, idx in entries:
                    obna_matches.append((header, text, line_num, idx))

        if not obna_matches:
            if args.verbose:
                print(f"  #{obj_id:04d}: OBNA not found in swedish.txt (may be untranslated)")
            continue

        for header, text, line_num, idx in obna_matches:
            encoded = encode_swedish(text)
            current_len = scumm_byte_len(encoded)
            stripped = strip_trailing_at(encoded)
            stripped_len = scumm_byte_len(stripped)

            # The required buffer is the max of:
            # - The English OBNA length (original buffer size)
            # - The longest replacement name (must fit)
            # We use the English buffer as the baseline since the SE writes in-place
            english_buffer = obna_len

            if required_buffer > english_buffer:
                # Replacement is longer than original OBNA — original game has this covered
                # via @ padding. Use the English buffer size.
                pass

            target_len = english_buffer  # must match original buffer

            if current_len == target_len:
                if args.verbose:
                    print(f"  #{obj_id:04d} L{line_num}: OK ({current_len} bytes)")
                continue

            needs_change += 1
            diff = target_len - current_len

            # Check if the stripped Swedish name is too long for the buffer
            if stripped_len > target_len:
                problems.append({
                    "obj_id": obj_id,
                    "line_num": line_num,
                    "header": header,
                    "swedish": stripped,
                    "swedish_len": stripped_len,
                    "buffer": target_len,
                    "overflow": stripped_len - target_len,
                })
                print(f"  #{obj_id:04d} L{line_num}: OVERFLOW! Swedish ({stripped_len}) > buffer ({target_len}) by {stripped_len - target_len}")
                print(f"    EN OBNA: {obna!r}")
                print(f"    SV text: {stripped!r}")
                longest_repl = max(data["replacements"], key=lambda r: r["len"])
                print(f"    Longest replacement: {longest_repl['name']!r} ({longest_repl['len']})")
            elif diff > 0:
                print(f"  #{obj_id:04d} L{line_num}: needs {diff} more @ (current {current_len}, need {target_len})")
                if args.verbose:
                    print(f"    SV: {encoded!r}")
            elif diff < 0:
                print(f"  #{obj_id:04d} L{line_num}: has {-diff} excess @ (current {current_len}, need {target_len})")

            if args.apply:
                # Apply padding
                new_text = pad_to_length(stripped, target_len)
                # If still too long (overflow), report but don't truncate
                if scumm_byte_len(new_text) > target_len:
                    print(f"    WARNING: cannot auto-pad — Swedish name overflows buffer. Shorten manually.")
                else:
                    # Reconstruct with opcode prefix if present
                    orig_text = text
                    prefix = ""
                    if orig_text.startswith('(') and ')' in orig_text:
                        prefix = orig_text[:orig_text.index(')') + 1]
                    # Write back the encoded text (with SCUMM escapes, not UTF-8)
                    # Actually, swedish.txt uses UTF-8, so we need to keep it in UTF-8
                    # but with @ padding added
                    stripped_utf8 = text
                    if stripped_utf8.startswith('(') and ')' in stripped_utf8:
                        stripped_utf8 = stripped_utf8[stripped_utf8.index(')') + 1:]
                    stripped_utf8 = stripped_utf8.rstrip('@')
                    # Calculate how many @ we need in UTF-8 terms
                    encoded_stripped = encode_swedish(stripped_utf8)
                    needed_at = target_len - scumm_byte_len(encoded_stripped)
                    if needed_at > 0:
                        new_utf8 = prefix + stripped_utf8 + '@' * needed_at
                        swedish_lines[idx] = (header, new_utf8, line_num)

    print()
    print("=" * 70)
    print("SUMMARY")
    print("=" * 70)
    print(f"  Objects with replacements: {total_objects}")
    print(f"  Lines needing padding changes: {needs_change}")
    print(f"  Buffer overflows (need shorter Swedish): {len(problems)}")

    if problems:
        print()
        print("  OVERFLOWS requiring manual shortening:")
        for p in problems:
            print(f"    #{p['obj_id']:04d} L{p['line_num']}: {p['swedish']!r} overflows by {p['overflow']}")

    if args.apply and needs_change > 0:
        print()
        print(f"  Applying changes to {sv_path}...")
        with open(sv_path, 'w', encoding='utf-8') as f:
            for header, text, line_num in swedish_lines:
                f.write(f"{header}{text}\n")
        print(f"  Done. {needs_change} lines updated.")


if __name__ == "__main__":
    main()
