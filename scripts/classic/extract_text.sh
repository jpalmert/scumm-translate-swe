#!/usr/bin/env bash
# Extract all text from a classic SCUMM game into a translation file.
#
# Usage:
#   bash scripts/classic/extract_text.sh <game_id> <game_dir> <output_file>
#
# Arguments:
#   game_id     scummtr game identifier (e.g. monkeycd, monkey2, indy4, sam)
#               Run: tools/bin/scummtr -L  to list all supported IDs
#   game_dir    Directory containing the game resource files (MONKEY.000 etc.)
#   output_file Path to write the extracted text (e.g. games/monkey1/text/translation.txt)
#
# Examples:
#   bash scripts/classic/extract_text.sh monkeycd ~/games/monkey1/ games/monkey1/text/translation.txt
#   bash scripts/classic/extract_text.sh monkey2  ~/games/monkey2/ games/monkey2/text/translation.txt

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCUMMTR="$REPO_ROOT/tools/bin/scummtr"

if [ $# -lt 3 ]; then
    echo "Usage: $0 <game_id> <game_dir> <output_file>"
    echo "       Run '$SCUMMTR -L' to list supported game IDs"
    exit 1
fi

GAME_ID="$1"
GAME_DIR="$2"
OUTPUT_FILE="$3"

if [ ! -f "$SCUMMTR" ]; then
    echo "ERROR: scummtr not found at $SCUMMTR"
    echo "       Run: bash scripts/install_deps.sh"
    exit 1
fi

if [ ! -d "$GAME_DIR" ]; then
    echo "ERROR: game directory not found: $GAME_DIR"
    exit 1
fi

mkdir -p "$(dirname "$OUTPUT_FILE")"

echo "Extracting text from '$GAME_ID' at $GAME_DIR ..."
"$SCUMMTR" -g "$GAME_ID" -cwh -A aov -p "$GAME_DIR" -of "$OUTPUT_FILE"

echo "Done. Text written to: $OUTPUT_FILE"
echo ""
echo "Line count: $(wc -l < "$OUTPUT_FILE")"
echo "Next step: translate the file, then run scripts/classic/inject_text.sh"
