#!/usr/bin/env bash
# Extract all translatable text from an SE PAK into JSON files ready for translation.
#
# Usage:
#   bash scripts/se/extract_for_translation.sh <game> <pak_file> <output_dir>
#
# Example:
#   bash scripts/se/extract_for_translation.sh 1 \
#     "/path/to/Monkey1.pak" \
#     games/monkey1/se_translations/

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MISE_DIR="$REPO_ROOT/tools/mise"
PYTHON="${REPO_ROOT}/.venv/bin/python3"

if [ $# -lt 3 ]; then
    echo "Usage: $0 <game> <pak_file> <output_dir>"
    exit 1
fi

GAME="$1"
PAK_FILE="$2"
OUTPUT_DIR="$3"

WORK_DIR="$(mktemp -d)"
mkdir -p "$OUTPUT_DIR"

echo "=== Extracting PAK ==="
"$PYTHON" "$MISE_DIR/pak.py" extract "$PAK_FILE" "$WORK_DIR" "$GAME"

echo ""
echo "=== Extracting text to JSON ==="

extract_info() {
    local info_name="$1"
    local json_name="$2"
    local info_file
    info_file="$(find "$WORK_DIR" -name "$info_name" -print -quit)"

    if [ -z "$info_file" ]; then
        echo "  WARNING: $info_name not found"
        return
    fi

    echo "  extracting $info_name -> $json_name"
    "$PYTHON" "$MISE_DIR/text.py" extract "$info_file" "$OUTPUT_DIR/$json_name"
}

if [ "$GAME" = "1" ]; then
    extract_info "speech.info"  "speech.json"
    extract_info "uiText.info"  "uitext.json"
else
    extract_info "fr.speech.info" "speech.json"
    extract_info "fr.uitext.info" "uitext.json"
fi

rm -rf "$WORK_DIR"

echo ""
echo "=== Done ==="
echo "JSON files written to: $OUTPUT_DIR"
echo ""
echo "Fill in the 'translation' field in each JSON entry, then run:"
echo "  bash scripts/se/build.sh $GAME <pak_file> $OUTPUT_DIR <output_pak>"
