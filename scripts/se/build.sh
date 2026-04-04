#!/usr/bin/env bash
# Full Special Edition translation build pipeline.
# Extracts the PAK, injects translated text and fonts, repacks to a new PAK.
#
# Usage:
#   bash scripts/se/build.sh <game> <original_pak> <translations_dir> <output_pak>
#
# Arguments:
#   game             1 for MI:SE, 2 for MI2:SE
#   original_pak     Path to original Monkey1.pak or Monkey2.pak from SE install
#   translations_dir Directory containing translated .json files
#                    (output of tools/mise/text.py extract, with translations filled in)
#   output_pak       Path to write the modified .pak file
#
# The translations_dir should contain JSON files named after the .info files:
#   speech.json       (from speech.info)
#   uitext.json       (from uiText.info or fr.uitext.info)
#
# Font modifications (optional):
#   If translations_dir/font_glyphs.png exists, it will be used to expand the font.
#   Alongside it, translations_dir/font_map.txt must contain the --map argument.
#
# IMPORTANT: The translated game requires the SE game language to be set to
#            French in order to display the custom translation.
#
# Example:
#   bash scripts/se/build.sh 1 \
#     "/path/to/SteamApps/Monkey Island SE/Monkey1.pak" \
#     games/monkey1/se_translations/ \
#     games/monkey1/patches/Monkey1_translated.pak

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MISE_DIR="$REPO_ROOT/tools/mise"
PYTHON="${REPO_ROOT}/.venv/bin/python3"

if [ $# -lt 4 ]; then
    echo "Usage: $0 <game> <original_pak> <translations_dir> <output_pak>"
    exit 1
fi

GAME="$1"
ORIGINAL_PAK="$2"
TRANSLATIONS_DIR="$3"
OUTPUT_PAK="$4"

if [ ! -f "$PYTHON" ]; then
    echo "ERROR: Python venv not found. Run: bash scripts/install_deps.sh"
    exit 1
fi

if [ ! -f "$ORIGINAL_PAK" ]; then
    echo "ERROR: original PAK not found: $ORIGINAL_PAK"
    exit 1
fi

WORK_DIR="$(mktemp -d)"
EXTRACTED_DIR="$WORK_DIR/extracted"
MODIFIED_DIR="$WORK_DIR/modified"

echo "=== Step 1: Extract PAK ==="
"$PYTHON" "$MISE_DIR/pak.py" extract "$ORIGINAL_PAK" "$EXTRACTED_DIR" "$GAME"

echo ""
echo "=== Step 2: Copy extracted files to work area ==="
cp -r "$EXTRACTED_DIR/." "$MODIFIED_DIR/"

echo ""
echo "=== Step 3: Inject translated text ==="

inject_info() {
    local info_name="$1"
    local json_name="$2"
    local info_file
    info_file="$(find "$MODIFIED_DIR" -name "$info_name" -print -quit)"

    if [ -z "$info_file" ]; then
        echo "  WARNING: $info_name not found in PAK, skipping"
        return
    fi

    local json_file="$TRANSLATIONS_DIR/$json_name"
    if [ ! -f "$json_file" ]; then
        echo "  WARNING: $json_file not found, skipping $info_name"
        return
    fi

    echo "  injecting $json_name -> $info_name"
    "$PYTHON" "$MISE_DIR/text.py" inject "$info_file" "$json_file" "$info_file"
}

if [ "$GAME" = "1" ]; then
    inject_info "speech.info"  "speech.json"
    inject_info "uiText.info"  "uitext.json"
else
    inject_info "fr.speech.info" "speech.json"
    inject_info "fr.uitext.info" "uitext.json"
fi

echo ""
echo "=== Step 4: Patch fonts (if glyphs provided) ==="

GLYPH_PNG="$TRANSLATIONS_DIR/font_glyphs.png"
FONT_MAP="$TRANSLATIONS_DIR/font_map.txt"

if [ -f "$GLYPH_PNG" ] && [ -f "$FONT_MAP" ]; then
    FONT_MAP_STR="$(cat "$FONT_MAP")"
    FONT_FILE="$(find "$MODIFIED_DIR" -name "*.font" -print -quit)"
    FONT_PNG="$(find "$MODIFIED_DIR" -name "*.font.png" -o -name "font*.png" | head -1)"

    if [ -n "$FONT_FILE" ] && [ -n "$FONT_PNG" ]; then
        echo "  expanding font: $FONT_FILE"
        "$PYTHON" "$MISE_DIR/font.py" add-glyphs \
            --font "$FONT_FILE" \
            --png  "$FONT_PNG" \
            --glyphs "$GLYPH_PNG" \
            --map    "$FONT_MAP_STR" \
            --out-font "$FONT_FILE" \
            --out-png  "$FONT_PNG"
    else
        echo "  WARNING: font file or font PNG not found in PAK, skipping font patch"
    fi
else
    echo "  no font glyphs provided ($GLYPH_PNG), skipping font step"
fi

echo ""
echo "=== Step 5: Repack PAK ==="
mkdir -p "$(dirname "$OUTPUT_PAK")"
"$PYTHON" "$MISE_DIR/pak.py" repack "$MODIFIED_DIR" "$OUTPUT_PAK" "$ORIGINAL_PAK" "$GAME"

echo ""
echo "=== Cleaning up ==="
rm -rf "$WORK_DIR"

echo ""
echo "=== Done ==="
echo "Output: $OUTPUT_PAK"
echo ""
echo "To use: replace Monkey1.pak (or Monkey2.pak) in your SE install with this file."
echo "        Set game language to French to see your translation."
