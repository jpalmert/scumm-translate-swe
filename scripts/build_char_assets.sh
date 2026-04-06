#!/usr/bin/env bash
# build_char_assets.sh — Produce the Swedish CHAR block binaries embedded in the patcher.
#
# This script is part of the dist build pipeline. Run it (or let build_patcher.sh
# call it) before `go build` so the //go:embed directives in charset.go can find
# the generated .bin files.
#
# How it works
# ------------
# For each classic CHAR block (0001, 0002, 0003, 0004, 0006):
#   1. Take the original English CHAR block from assets/charset/english/CHAR_NNNN.
#      These are extracted from the SE's embedded MONKEY1.001 via scummrp and are
#      the authoritative English glyph data for the monkeycdalt game ID.
#   2. Import the Swedish glyph BMP (internal/charset/bitmaps/CHAR_NNNN_swedish.bmp)
#      into a copy of the English block using scummfont.
#      scummfont replaces glyph pixel data in-place; the CHAR block structure is
#      preserved and size fields are updated automatically.
#   3. Write the result to internal/charset/assets/char_NNNN_patched.bin, which is
#      then embedded by Go at compile time.
#
# Source files (committed to git):
#   assets/charset/english/CHAR_NNNN       — original English CHAR blocks (binary)
#   internal/charset/bitmaps/CHAR_NNNN_swedish.bmp  — Swedish glyph bitmaps
#
# Generated files (gitignored, produced by this script):
#   internal/charset/assets/char_NNNN_patched.bin
#
# To update the Swedish glyphs:
#   1. Edit the relevant BMP in internal/charset/bitmaps/
#   2. Run this script
#   3. Commit both the BMP and the regenerated .bin file
#      (or just the BMP if you're letting build_patcher.sh regenerate the bins)
#
# Usage (from repo root):
#   bash scripts/build_char_assets.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Select the scummfont binary for the current platform.
if [[ "$(uname)" == "Darwin" ]]; then
    BIN="$REPO_ROOT/bin/darwin/scummfont"
else
    BIN="$REPO_ROOT/bin/linux/scummfont"
fi

if [[ ! -x "$BIN" ]]; then
    echo "ERROR: $BIN not found. Run bash scripts/install_deps.sh first." >&2
    exit 1
fi

ENGLISH="$REPO_ROOT/assets/charset/english"
BITMAPS="$REPO_ROOT/internal/charset/bitmaps"
ASSETS="$REPO_ROOT/internal/charset/assets"
TMPDIR_WORK="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_WORK"' EXIT

echo "=== Building charset assets ==="

for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    # Derive output filename: CHAR_0001 → char_0001_patched.bin
    lower="char_$(echo "${n#CHAR_}" )_patched.bin"

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
    echo "  $n -> $lower ($(wc -c < "$out" | tr -d ' ') bytes)"
done

echo ""
echo "Done. Run 'go build ./cmd/patcher' to produce the patcher binary."
