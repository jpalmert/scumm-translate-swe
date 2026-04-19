#!/usr/bin/env python3
"""
Calculate and apply @ padding for Swedish object names.

Reads dynamic_names.json (OBNA line -> replacement lines mapping) and
swedish.txt. For each OBNA, finds the longest Swedish replacement
(in SCUMM bytes) and pads the OBNA to at least that length.

Usage:
    python3 tools/calc_padding.py [--apply] [--json PATH] [--translation PATH]
"""

import argparse
import json
import os
import sys

SWEDISH_CHAR_MAP = {
    'Å': '\\091', 'Ä': '\\092', 'Ö': '\\093',
    'å': '\\123', 'ä': '\\124', 'ö': '\\125',
    'é': '\\130', 'ê': '\\136', '™': '\\153',
}


def encode_swedish(text):
    for char, esc in SWEDISH_CHAR_MAP.items():
        text = text.replace(char, esc)
    if text.startswith('(') and ')' in text:
        text = text[text.index(')') + 1:]
    return text


def scumm_byte_len(text):
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
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true", help="Apply padding in place")
    parser.add_argument("--json", default=None)
    parser.add_argument("--translation", default=None)
    args = parser.parse_args()

    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    json_path = args.json or os.path.join(repo, "games", "monkey1", "gen", "dynamic_names.json")
    sv_path = args.translation or os.path.join(repo, "games", "monkey1", "translation", "swedish.txt")

    if not os.path.isfile(json_path):
        print(f"Error: {json_path} not found. Run scripts/extract.sh first.", file=sys.stderr)
        sys.exit(1)

    with open(json_path) as f:
        mapping = json.load(f)

    with open(sv_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()

    def get_text(lineno):
        """Get the text portion of a 1-based line number."""
        raw = lines[lineno - 1].rstrip('\r\n')
        if raw.startswith('[') and ']' in raw:
            return raw[raw.index(']') + 1:]
        return raw

    changes = {}
    pad_count = 0

    for obna_lineno_str, repl_linenos in sorted(mapping.get("replacements", {}).items(), key=lambda x: int(x[0])):
        obna_lineno = int(obna_lineno_str)
        obna_text = get_text(obna_lineno)
        obna_encoded = encode_swedish(obna_text)
        obna_stripped = obna_encoded.rstrip('@')
        obna_len = scumm_byte_len(obna_stripped)

        # Find longest replacement
        max_repl_len = 0
        for rln in repl_linenos:
            repl_text = get_text(rln)
            repl_encoded = encode_swedish(repl_text)
            repl_stripped = repl_encoded.rstrip('@')
            rlen = scumm_byte_len(repl_stripped)
            if rlen > max_repl_len:
                max_repl_len = rlen

        required = max(obna_len, max_repl_len)
        current = scumm_byte_len(obna_encoded)

        if current >= required:
            continue  # existing padding (if any) is already sufficient

        pad_count += 1
        extra_at = required - current

        # Append additional @ padding without touching existing content
        raw = lines[obna_lineno - 1].rstrip('\r\n')
        new_line = raw + '@' * extra_at + '\n'
        changes[obna_lineno - 1] = new_line

        print(f"  L{obna_lineno}: pad to {required} (currently {current}, adding {extra_at} @, longest repl: {max_repl_len}, obna text: {obna_len})")

    print(f"\nLines needing padding: {pad_count}")

    if args.apply and changes:
        for idx, new_line in changes.items():
            lines[idx] = new_line
        with open(sv_path, 'w', encoding='utf-8') as f:
            f.writelines(lines)
        print(f"Applied {len(changes)} changes to {sv_path}")


if __name__ == "__main__":
    main()
