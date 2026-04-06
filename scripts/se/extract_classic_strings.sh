#!/usr/bin/env bash
# extract_classic_strings.sh — Extract dialog strings from MI1SE for translation work
#
# What this does:
#   Extracts all English dialog strings and writes them to a text file,
#   ready for translation. Each string is preceded by a [room:type#id] header.
#
# Two input modes:
#   1. PAK file (SE version):
#      The script extracts the embedded classic SCUMM files from Monkey1.pak,
#      then runs scummtr on them.
#
#   2. Classic files directory (classic CD version, or manually extracted SE files):
#      If you already have MONKEY1.000 and MONKEY1.001 (or monkey1.000/.001),
#      pass the directory containing them directly — no PAK needed.
#
# Output:
#   assets/classic/english_strings.txt — English strings, one per line,
#                                        with [room:type#id] context headers (gitignored)
#
# Prerequisites:
#   - bin/scummtr must exist (run: bash scripts/install_deps.sh)
#   - game/monkey1/Monkey1.pak OR a directory with MONKEY1.000 + MONKEY1.001
#
# Usage:
#   bash scripts/se/extract_classic_strings.sh
#   bash scripts/se/extract_classic_strings.sh /path/to/Monkey1.pak
#   bash scripts/se/extract_classic_strings.sh /path/to/classic/files/

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
case "$(uname -s)" in
  Darwin) SCUMMTR="$REPO_ROOT/bin/darwin/scummtr" ;;
  *)      SCUMMTR="$REPO_ROOT/bin/linux/scummtr"  ;;
esac
PAKPY="$REPO_ROOT/tools/pak.py"
WORK_DIR="$(mktemp -d)"
OUT_DIR="$REPO_ROOT/assets/classic"
OUT_FILE="$OUT_DIR/english_strings.txt"

trap 'rm -rf "$WORK_DIR"' EXIT

# --- Validate prerequisites ---

if [ ! -f "$SCUMMTR" ]; then
    echo "ERROR: scummtr not found at $SCUMMTR"
    echo "       Run: bash scripts/install_deps.sh"
    exit 1
fi

# --- Determine input mode ---

INPUT="${1:-$REPO_ROOT/game/monkey1/Monkey1.pak}"
CLASSIC_DIR=""

if [ -d "$INPUT" ]; then
    # Directory mode: user provided a directory containing MONKEY1.000/.001
    echo "==> Using classic files directory: $INPUT"

    # Accept upper or lowercase filenames
    if [ -f "$INPUT/MONKEY1.000" ] || [ -f "$INPUT/monkey1.000" ]; then
        mkdir -p "$WORK_DIR/classic"
        for f in MONKEY1.000 MONKEY1.001 monkey1.000 monkey1.001; do
            [ -f "$INPUT/$f" ] && cp "$INPUT/$f" "$WORK_DIR/classic/${f^^}"
        done
        CLASSIC_DIR="$WORK_DIR/classic"
    else
        echo "ERROR: Directory does not contain MONKEY1.000 / MONKEY1.001: $INPUT"
        exit 1
    fi
elif [ -f "$INPUT" ]; then
    # PAK mode: extract embedded classic files from the PAK
    echo "==> Extracting PAK: $INPUT"
    python3 "$PAKPY" extract "$INPUT" "$WORK_DIR/pak"

    echo "==> Copying embedded classic data"
    mkdir -p "$WORK_DIR/classic"
    cp "$WORK_DIR/pak/classic/en/monkey1.000" "$WORK_DIR/classic/MONKEY1.000"
    cp "$WORK_DIR/pak/classic/en/monkey1.001" "$WORK_DIR/classic/MONKEY1.001"
    CLASSIC_DIR="$WORK_DIR/classic"
else
    echo "ERROR: Not found: $INPUT"
    echo "       Pass a Monkey1.pak file or a directory containing MONKEY1.000 + MONKEY1.001"
    exit 1
fi

# --- Extract strings with scummtr ---
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
"$SCUMMTR" -g monkeycdalt -p "$CLASSIC_DIR" -cwh -A aov -o -f "$OUT_FILE"

LINE_COUNT=$(wc -l < "$OUT_FILE")
echo ""
echo "==> Done: $LINE_COUNT lines written to $OUT_FILE"
echo "    (2 are scummtr comment headers; real string count is $((LINE_COUNT - 2)))"
