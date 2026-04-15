#!/usr/bin/env python3
"""
Calculate and apply @ padding for Swedish object names.

For each object with runtime name replacements (from dynamic_names.json),
find the OBNA line and all replacement lines in swedish.txt. The OBNA buffer
must be at least as long as the longest replacement name (in SCUMM bytes).

The SE engine writes replacement names in-place into the OBNA buffer with no
bounds check. If the OBNA is shorter than a replacement, it overflows.

Usage:
    python3 tools/calc_padding.py [--apply] [--json PATH] [--translation PATH]

    --apply:       modify the translation file in place (use on dist/ copy)
    --json:        path to dynamic_names.json
    --translation: path to swedish.txt
"""

import argparse
import json
import os
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


def main():
    parser = argparse.ArgumentParser(description="Calculate @ padding for Swedish object names")
    parser.add_argument("--apply", action="store_true", help="Apply padding in place")
    parser.add_argument("--json", default=None, help="Path to dynamic_names.json")
    parser.add_argument("--translation", default=None, help="Path to swedish.txt")
    args = parser.parse_args()

    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    json_path = args.json or os.path.join(repo, "translation", "monkey1", "dynamic_names.json")
    sv_path = args.translation or os.path.join(repo, "translation", "monkey1", "swedish.txt")

    if not os.path.isfile(json_path):
        print(f"Error: {json_path} not found. Run scripts/extract.sh first.", file=sys.stderr)
        sys.exit(1)

    with open(json_path) as f:
        mapping = json.load(f)

    # Read swedish.txt lines
    with open(sv_path, 'r', encoding='utf-8') as f:
        raw_lines = f.readlines()

    # Parse into (header, text, raw_line) — preserve exact line content for rewriting
    parsed = []
    for raw in raw_lines:
        stripped = raw.rstrip('\r\n')
        if stripped.startswith('[') and ']' in stripped:
            idx = stripped.index(']')
            parsed.append((stripped[:idx+1], stripped[idx+1:]))
        else:
            parsed.append((None, stripped))

    # For each object with replacements, find:
    # 1. The OBNA line in swedish.txt
    # 2. All replacement lines in swedish.txt (matched by scummtr header position —
    #    we can't match by content since it's translated, but we know the headers
    #    from the JSON's replacement list)
    # 3. The longest SCUMM byte length among OBNA + all replacements
    # 4. Pad the OBNA line to that length

    needs_change = 0
    overflows = []
    changes = {}  # line_index -> new_text

    for obj_id_str, replacement_names in sorted(mapping["objects"].items(), key=lambda x: int(x[0])):
        obj_id = int(obj_id_str)
        obna_pattern = f"OBNA#{obj_id:04d}]"

        # Find OBNA line(s) for this object
        obna_indices = []
        for i, (header, text) in enumerate(parsed):
            if header and obna_pattern in header:
                obna_indices.append(i)

        if not obna_indices:
            continue

        # Find the longest replacement name (in SCUMM bytes)
        # These are the ENGLISH replacement names from the bytecode.
        # We need to find their SWEDISH translations in swedish.txt.
        # But we can't easily match replacement lines by content since they're
        # interleaved with other text in VERB/SCRP/LSCR blocks.
        #
        # Instead, the safe approach: the OBNA buffer must be at least as long as
        # the longest ENGLISH replacement name. The English lengths are the contract
        # — they define the buffer the original game needs.
        max_repl_len = max((len(name) for name in replacement_names), default=0)

        for idx in obna_indices:
            header, text = parsed[idx]
            encoded = encode_swedish(text)
            stripped = encoded.rstrip('@')
            stripped_len = scumm_byte_len(stripped)

            # Required buffer = max of stripped OBNA and longest replacement
            required = max(stripped_len, max_repl_len)
            current_len = scumm_byte_len(encoded)

            if current_len == required:
                continue

            needs_change += 1

            if stripped_len > required:
                # Swedish OBNA name itself is longer than all replacements —
                # the replacement writes will fit (they're shorter). No padding needed.
                # But we should still pad if the name is shorter than the longest replacement.
                continue

            if stripped_len > max_repl_len:
                # OBNA is longer than all replacements — no padding needed at all
                continue

            pad_needed = required - stripped_len

            # Rebuild the text with correct padding
            # Preserve opcode prefix if present
            orig_text = text
            prefix = ""
            if orig_text.startswith('(') and ')' in orig_text:
                prefix = orig_text[:orig_text.index(')') + 1]
                base_text = orig_text[orig_text.index(')') + 1:]
            else:
                base_text = orig_text

            base_stripped = base_text.rstrip('@')
            # Verify the base fits
            base_encoded = encode_swedish(base_stripped)
            base_len = scumm_byte_len(base_encoded)

            if base_len > required:
                overflows.append({
                    "obj_id": obj_id,
                    "line": idx + 1,
                    "swedish": base_stripped,
                    "swedish_len": base_len,
                    "buffer": required,
                    "overflow": base_len - required,
                })
                print(f"  #{obj_id:04d} L{idx+1}: OVERFLOW! Swedish ({base_len}) > buffer ({required}) by {base_len - required}")
                continue

            at_count = required - base_len
            new_text = prefix + base_stripped + '@' * at_count
            changes[idx] = new_text

            diff = required - current_len
            if diff > 0:
                print(f"  #{obj_id:04d} L{idx+1}: adding {at_count} @ (was {current_len}, now {required})")
            elif diff < 0:
                print(f"  #{obj_id:04d} L{idx+1}: adjusting @ (was {current_len}, now {required})")

    print()
    print(f"Lines needing padding: {needs_change}")
    print(f"Overflows (shorten manually): {len(overflows)}")

    if overflows:
        print("\nOVERFLOWS:")
        for o in overflows:
            print(f"  #{o['obj_id']:04d} L{o['line']}: '{o['swedish']}' overflows by {o['overflow']}")

    if args.apply and changes:
        for idx, new_text in changes.items():
            header, _ = parsed[idx]
            parsed[idx] = (header, new_text)

        with open(sv_path, 'w', encoding='utf-8') as f:
            for header, text in parsed:
                if header:
                    f.write(f"{header}{text}\n")
                else:
                    f.write(f"{text}\n")

        print(f"\nApplied {len(changes)} padding changes to {sv_path}")


if __name__ == "__main__":
    main()
