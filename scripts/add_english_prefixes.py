#!/usr/bin/env python3
"""
Add [E] prefix to all untranslated lines in swedish.txt from english.txt
This allows git diff to show English and Swedish side-by-side.
"""

import sys

def main():
    english_path = "game/monkey1/gen/strings/english.txt"
    swedish_path = "translation/monkey1/swedish.txt"

    print(f"Reading {english_path}...")
    with open(english_path, 'r', encoding='utf-8') as f:
        english_lines = f.readlines()

    print(f"Reading {swedish_path}...")
    with open(swedish_path, 'r', encoding='utf-8') as f:
        swedish_lines = f.readlines()

    if len(english_lines) != len(swedish_lines):
        print(f"ERROR: Line count mismatch!")
        print(f"  English: {len(english_lines)} lines")
        print(f"  Swedish: {len(swedish_lines)} lines")
        sys.exit(1)

    modified_count = 0
    output_lines = []

    for i, (eng, swe) in enumerate(zip(english_lines, swedish_lines), 1):
        # If Swedish line is blank/empty, populate with [E] + English
        # If Swedish line already has [E] prefix, keep it
        # If Swedish line has translation content, keep it

        swe_stripped = swe.strip()

        if not swe_stripped:
            # Blank line - populate with [E] + English
            if eng.endswith('\n'):
                output_lines.append(f"[E]{eng}")
            else:
                output_lines.append(f"[E]{eng}\n")
            modified_count += 1
        elif swe.startswith('[E]'):
            # Already has [E] prefix, keep it
            output_lines.append(swe)
        else:
            # Has Swedish translation, keep it
            output_lines.append(swe)

    print(f"Writing {swedish_path}...")
    with open(swedish_path, 'w', encoding='utf-8') as f:
        f.writelines(output_lines)

    print(f"Done! Added [E] prefix to {modified_count} untranslated lines.")

if __name__ == '__main__':
    main()
