#!/usr/bin/env bash
# build_char_assets.sh — Import Swedish glyph BMPs into CHAR block templates
#
# Reads source BMPs from  internal/charset/bitmaps/CHAR_NNNN_swedish.bmp
# Reads English templates from  internal/charset/english/CHAR_NNNN
# Writes patched CHAR blocks to  internal/charset/assets/char_NNNN_patched.bin
#
# Run this whenever you modify a BMP, then commit both the BMP and the
# regenerated .bin file.
#
# Usage (from repo root):
#   bash scripts/build_char_assets.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$REPO_ROOT/bin/linux/scummfont"

if [[ "$(uname)" == "Darwin" ]]; then
    BIN="$REPO_ROOT/bin/darwin/scummfont"
fi

if [[ ! -x "$BIN" ]]; then
    echo "ERROR: $BIN not found. Run bash scripts/install_deps.sh first." >&2
    exit 1
fi

BITMAPS="$REPO_ROOT/internal/charset/bitmaps"
ENGLISH="$REPO_ROOT/internal/charset/english"
ASSETS="$REPO_ROOT/internal/charset/assets"
TMPDIR_WORK="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_WORK"' EXIT

echo "=== Building charset assets ==="

for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    lower="$(echo "$n" | tr '[:upper:]' '[:lower:]' | sed 's/char_/char_/' | sed 's/$/_patched.bin/')"
    bmp="$BITMAPS/${n}_swedish.bmp"
    template="$ENGLISH/$n"
    out="$ASSETS/$lower"
    work="$TMPDIR_WORK/$n"

    if [[ ! -f "$bmp" ]]; then
        echo "  SKIP $n: $bmp not found"
        continue
    fi
    if [[ ! -f "$template" ]]; then
        echo "  SKIP $n: $template not found"
        continue
    fi

    cp "$template" "$work"
    "$BIN" i "$work" "$bmp"
    cp "$work" "$out"
    echo "  $n -> $lower ($(wc -c < "$out") bytes)"
done

echo ""
echo "Done. Commit internal/charset/assets/ if any files changed."
