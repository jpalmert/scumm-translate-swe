#!/usr/bin/env python3
"""
merge_translation.py — Merge Swedish text into the English scummtr file structure.

Reads english.txt (the structural template with @-padding and backslash-NNN control codes)
and monkey1.txt.old (Swedish text, no @-padding), and produces a merged monkey1.txt
that has the English control structure with Swedish translatable text.

Usage:
    python3 tools/merge_translation.py \
        game/monkey1/gen/strings/english.txt \
        translation/monkey1/monkey1.txt.old \
        translation/monkey1/monkey1.txt

The output file is suitable for direct injection via InjectTranslation (classic.go),
which uses non-raw mode (-ih).  Swedish chars (aaoAAOe) in the input are passed
through as-is; the encodeForScummtr step in classic.go converts them to escape codes
before handing off to scummtr.
"""

import re
import sys

# Splits a line's content (after the header) into alternating [delimiter, text] tokens.
# Delimiters: \NNN decimal escapes, ^ (newline/pause), ` (backtick control), @+ (NUL padding).
# Text segments are what gets translated; delimiters are preserved from the English template.
CTRL_RE = re.compile(r'(\\[0-9]{3}|\^|`|@+)')


def tokenize(content):
    """Return list of tokens, starting and ending with text (possibly empty)."""
    return CTRL_RE.split(content)


def text_segments(tokens):
    """Extract every other token starting at index 0 (the text parts)."""
    return tokens[0::2]


def strip_trailing_at(tokens):
    """
    If tokens end with ['@@@...', ''], strip that pair and return (stripped, at_string).
    Otherwise return (tokens, None).
    """
    if (len(tokens) >= 3
            and tokens[-1] == ''
            and tokens[-2]
            and all(c == '@' for c in tokens[-2])):
        return tokens[:-2], tokens[-2]
    return tokens, None


def merge_line(en_content, sv_content):
    """
    Replace text segments in en_content with those from sv_content.

    @-padding handling (fixed-width slots): trailing @-runs are stripped from
    both sides before comparing text segment counts, then re-added using the
    English slot size (so Swedish text is padded to match the game's fixed slot).

    Returns (merged_content, fallback_used) where fallback_used is True if the
    Swedish line was used as-is due to structural mismatch.
    """
    en_tokens = tokenize(en_content)
    sv_tokens = tokenize(sv_content)

    # Strip trailing @ padding from both sides independently.
    en_core, en_at = strip_trailing_at(en_tokens)
    sv_core, sv_at = strip_trailing_at(sv_tokens)

    en_texts = text_segments(en_core)
    sv_texts = text_segments(sv_core)

    en_delims = en_core[1::2]
    sv_delims = sv_core[1::2]

    if len(en_texts) != len(sv_texts) or en_delims != sv_delims:
        # Structural mismatch: different segment count or delimiter sequence.
        # Fall back to Swedish content as-is.
        return sv_content, True

    # Rebuild using English delimiter structure, Swedish text.
    result = []
    for i, tok in enumerate(en_core):
        if i % 2 == 0:
            result.append(sv_texts[i // 2])
        else:
            result.append(tok)

    if en_at is not None:
        # Re-pad to English slot size regardless of whether Swedish had @ padding.
        en_slot_text = en_texts[-1]
        sv_slot_text = sv_texts[-1]
        slot_size = len(en_slot_text) + len(en_at)
        if len(sv_slot_text) > slot_size:
            # Swedish text overflows slot — use Swedish as-is (scummtr will handle it).
            return sv_content, True
        result.append('@' * (slot_size - len(sv_slot_text)))
        result.append('')

    return ''.join(result), False


def parse_header(line):
    """Return (header, content) or (None, line) for comment/blank lines."""
    if line.startswith(';;') or not line.startswith('['):
        return None, line
    bracket = line.index(']')
    return line[:bracket + 1], line[bracket + 1:]


def main():
    if len(sys.argv) != 4:
        print(f"Usage: {sys.argv[0]} english.txt swedish_old.txt output.txt", file=sys.stderr)
        sys.exit(1)

    en_path, sv_path, out_path = sys.argv[1], sys.argv[2], sys.argv[3]

    with open(en_path, encoding='latin-1') as f:
        en_lines = f.read().splitlines()
    with open(sv_path, encoding='utf-8') as f:
        sv_lines = f.read().splitlines()

    if len(en_lines) != len(sv_lines):
        print(f"ERROR: line count mismatch: en={len(en_lines)} sv={len(sv_lines)}", file=sys.stderr)
        sys.exit(1)

    errors = 0
    out_lines = []

    for lineno, (en_line, sv_line) in enumerate(zip(en_lines, sv_lines), 1):
        en_header, en_content = parse_header(en_line)
        sv_header, sv_content = parse_header(sv_line)

        # Comments and blank lines: use the English line unchanged
        if en_header is None:
            out_lines.append(en_line)
            continue

        # Verify headers match
        if en_header != sv_header:
            print(f"ERROR line {lineno}: header mismatch: en={en_header!r} sv={sv_header!r}", file=sys.stderr)
            errors += 1
            out_lines.append(en_line)
            continue

        try:
            merged, fallback = merge_line(en_content, sv_content)
            if fallback:
                print(f"FALLBACK line {lineno} ({en_header}): structural mismatch, using Swedish line as-is",
                      file=sys.stderr)
                errors += 1
                out_lines.append(sv_line)
            else:
                out_lines.append(en_header + merged)
        except ValueError as e:
            print(f"ERROR line {lineno} ({en_header}): {e}", file=sys.stderr)
            errors += 1
            out_lines.append(sv_line)

    if errors:
        print(f"\n{errors} merge error(s) — output written with English fallback for failed lines.", file=sys.stderr)

    with open(out_path, 'w', encoding='utf-8') as f:
        f.write('\n'.join(out_lines) + '\n')

    print(f"Written {len(out_lines)} lines to {out_path} ({errors} errors)")


if __name__ == '__main__':
    main()
