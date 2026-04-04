#!/usr/bin/env bash
# Inject a translated text file back into classic SCUMM game resource files.
# Works on a COPY of the game files — never modify your originals directly.
#
# Usage:
#   bash scripts/classic/inject_text.sh <game_id> <game_dir> <input_file>
#
# Arguments:
#   game_id     scummtr game identifier (same as used for extraction)
#   game_dir    Directory containing the game resource files to modify IN PLACE
#               Point this at a COPY of your originals.
#   input_file  Translated text file (output of extract_text.sh, now translated)
#
# The text file must use the same flags (-cwh -A aov) as extraction.
# Swedish special characters must be replaced with SCUMM escape codes first.
# See games/monkey1/references/TRANSLATE_TABLE for the mapping.
#
# Examples:
#   bash scripts/classic/inject_text.sh monkeycd ~/games/monkey1_copy/ games/monkey1/text/translation.txt

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCUMMTR="$REPO_ROOT/tools/bin/scummtr"

if [ $# -lt 3 ]; then
    echo "Usage: $0 <game_id> <game_dir> <input_file>"
    exit 1
fi

GAME_ID="$1"
GAME_DIR="$2"
INPUT_FILE="$3"

if [ ! -f "$SCUMMTR" ]; then
    echo "ERROR: scummtr not found at $SCUMMTR"
    echo "       Run: bash scripts/install_deps.sh"
    exit 1
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "ERROR: input file not found: $INPUT_FILE"
    exit 1
fi

# Apply Swedish character code substitutions (see games/monkey1/references/TRANSLATE_TABLE)
TEMP_FILE="$(mktemp)"
iconv -f utf-8 -t iso-8859-1 "$INPUT_FILE" -o "$TEMP_FILE"
sed -i \
    's/Å/\\091/g;s/Ä/\\092/g;s/Ö/\\093/g' \
    "$TEMP_FILE"
sed -i \
    's/å/\\123/g;s/ä/\\124/g;s/ö/\\125/g;s/é/\\130/g' \
    "$TEMP_FILE"

echo "Injecting text into '$GAME_ID' at $GAME_DIR ..."
"$SCUMMTR" -g "$GAME_ID" -cwh -A aov -p "$GAME_DIR" -if "$TEMP_FILE"

rm -f "$TEMP_FILE"

echo "Done. Game files in $GAME_DIR have been modified."
echo "Next step: run scripts/classic/build_patch.sh to create distributable BPS patches."
