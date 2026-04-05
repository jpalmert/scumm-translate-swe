#!/usr/bin/env bash
# extract_classic_strings.sh — Extract dialog strings from the SE PAK for translation work
#
# What this does:
#   1. Extracts game/monkey1/Monkey1.pak into a temp directory
#   2. Copies the embedded classic/en/MONKEY1.000 + MONKEY1.001 files
#   3. Runs scummtr to dump all dialog strings to a text file
#
# The SE stores game dialog in embedded classic SCUMM files (not SE-specific .info files).
# This means the same scummtr workflow used for the classic CD version works here too.
# See docs/OPEN_QUESTIONS.md OQ-2 for the investigation that confirmed this.
#
# Output:
#   game/monkey1/text/se_english.txt — English strings, one per line,
#                                      with [room:type#id] context headers (gitignored)
#
# Prerequisites:
#   - bin/scummtr must exist (run: bash scripts/install_deps.sh)
#   - game/monkey1/Monkey1.pak must be present (gitignored — user provides)
#
# Usage:
#   bash scripts/se/extract_classic_strings.sh
#   bash scripts/se/extract_classic_strings.sh /path/to/Monkey1.pak

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PAK="${1:-$REPO_ROOT/game/monkey1/Monkey1.pak}"
case "$(uname -s)" in
  Darwin) SCUMMTR="$REPO_ROOT/bin/darwin/scummtr" ;;
  *)      SCUMMTR="$REPO_ROOT/bin/linux/scummtr"  ;;
esac
PAKPY="$REPO_ROOT/tools/pak.py"
WORK_DIR="$(mktemp -d)"
OUT_DIR="$REPO_ROOT/game/monkey1/text"
OUT_FILE="$OUT_DIR/se_english.txt"

trap 'rm -rf "$WORK_DIR"' EXIT

# --- Validate prerequisites ---

if [ ! -f "$SCUMMTR" ]; then
    echo "ERROR: scummtr not found at $SCUMMTR"
    echo "       Run: bash scripts/install_deps.sh"
    exit 1
fi

if [ ! -f "$PAK" ]; then
    echo "ERROR: PAK file not found: $PAK"
    echo "       Copy Monkey1.pak to game/monkey1/, or pass its path as an argument."
    exit 1
fi

# --- Step 1: Extract PAK ---

echo "==> Extracting PAK: $PAK"
python3 "$PAKPY" extract "$PAK" "$WORK_DIR/pak"

# The SE PAK embeds classic SCUMM data at classic/en/monkey1.000 + monkey1.001
# scummtr requires the files to be named MONKEY1.000 / MONKEY1.001 (uppercase)
# Game ID: monkeycdalt (scummtr's name for the MONKEY1.000 file variant)

echo "==> Copying embedded classic data"
mkdir -p "$WORK_DIR/classic"
cp "$WORK_DIR/pak/classic/en/monkey1.000" "$WORK_DIR/classic/MONKEY1.000"
cp "$WORK_DIR/pak/classic/en/monkey1.001" "$WORK_DIR/classic/MONKEY1.001"

# --- Step 2: Extract strings with scummtr ---
#
# Flags used:
#   -g monkeycdalt  game ID for MONKEY1.000 file layout
#   -p              path to game files
#   -c              convert characters to Windows-1252 (handles special chars)
#   -w              Windows CRLF line endings (scummtr standard)
#   -h              include [room:type#id] context header before each string
#   -A aov          protect actor/object/verb names from accidental corruption
#   -o              output mode (export strings FROM the game)
#   -f              output file path

echo "==> Extracting strings with scummtr"
mkdir -p "$OUT_DIR"
"$SCUMMTR" -g monkeycdalt -p "$WORK_DIR/classic" -cwh -A aov -o -f "$OUT_FILE"

LINE_COUNT=$(wc -l < "$OUT_FILE")
echo ""
echo "==> Done: $LINE_COUNT lines written to $OUT_FILE"
echo "    (2 are scummtr comment headers; real string count is $((LINE_COUNT - 2)))"
