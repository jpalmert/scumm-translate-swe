#!/usr/bin/env bash
# extract_char_bitmaps.sh — Export the English CHAR blocks as editable BMP files.
#
# This is a developer convenience script. Run it when you want to see what the
# original English glyphs look like — for example, as a reference while editing
# the Swedish BMP files in internal/charset/bitmaps/.
#
# Reads:   assets/charset/english/CHAR_NNNN   (committed English CHAR block templates)
# Writes:  assets/charset/english_bitmaps/CHAR_NNNN.bmp   (gitignored, local only)
#
# The output BMPs are not committed and are not needed for the build. They exist
# purely for visual reference.
#
# Usage (from repo root):
#   bash scripts/extract_char_bitmaps.sh

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
OUT_DIR="$REPO_ROOT/assets/charset/english_bitmaps"
mkdir -p "$OUT_DIR"

echo "=== Exporting English CHAR blocks to BMP ==="
echo "    Output: $OUT_DIR"
echo ""

for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    template="$ENGLISH/$n"
    bmp="$OUT_DIR/${n}.bmp"

    if [[ ! -f "$template" ]]; then
        echo "  SKIP $n: template not found"
        continue
    fi

    "$BIN" o "$template" "$bmp"
    echo "  $n -> ${n}.bmp"
done

echo ""
echo "Done. Open the BMPs in GIMP or similar to inspect English glyph layout."
echo "(These files are gitignored — for local reference only.)"
