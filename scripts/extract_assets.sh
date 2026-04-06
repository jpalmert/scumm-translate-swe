#!/usr/bin/env bash
# extract_assets.sh — Extract English charset assets and dialog strings from MI1 game files.
#
# Works from a directory containing the classic SCUMM data files.
# Accepts both naming conventions:
#   MONKEY1.000 / MONKEY1.001  — SE embedded classic (game ID: monkeycdalt)
#   MONKEY.000  / MONKEY.001   — Classic CD version  (game ID: monkeycd)
# Both upper and lowercase filenames are accepted.
#
# Reads:   game dir containing MONKEY1.000/001 or MONKEY.000/001
# Writes:  game/monkey1/gen/charset/english/CHAR_NNNN  — raw CHAR font blocks
#          game/monkey1/gen/charset/english_bitmaps/*.bmp — visual reference for editing Swedish glyphs
#          game/monkey1/gen/strings/english.txt          — dialog strings for translation
#
# All outputs live under game/ which is gitignored.
#
# Usage (from repo root):
#   bash scripts/extract_assets.sh [game_dir]
#   Default game_dir: game/monkey1/
#
# Prerequisites:
#   bash scripts/install_deps.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SCUMMRP="$REPO_ROOT/bin/linux/scummrp"
SCUMMFONT="$REPO_ROOT/bin/linux/scummfont"
SCUMMTR="$REPO_ROOT/bin/linux/scummtr"
if [[ "$(uname)" == "Darwin" ]]; then
    SCUMMRP="$REPO_ROOT/bin/darwin/scummrp"
    SCUMMFONT="$REPO_ROOT/bin/darwin/scummfont"
    SCUMMTR="$REPO_ROOT/bin/darwin/scummtr"
fi

for bin in "$SCUMMRP" "$SCUMMFONT" "$SCUMMTR"; do
    if [[ ! -x "$bin" ]]; then
        echo "ERROR: $(basename "$bin") not found. Run: bash scripts/install_deps.sh" >&2
        exit 1
    fi
done

INPUT_DIR="${1:-$REPO_ROOT/game/monkey1}"

if [[ ! -d "$INPUT_DIR" ]]; then
    echo "ERROR: directory not found: $INPUT_DIR" >&2
    exit 1
fi

# --- Detect game variant and normalise filenames into a temp work dir ---
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
GAME_DIR="$WORK/game"
mkdir -p "$GAME_DIR"

GAME_ID=""

# SE embedded classic: MONKEY1.000 / MONKEY1.001
for name in MONKEY1 monkey1; do
    if [[ -f "$INPUT_DIR/$name.000" && -f "$INPUT_DIR/$name.001" ]]; then
        cp "$INPUT_DIR/$name.000" "$GAME_DIR/MONKEY1.000"
        cp "$INPUT_DIR/$name.001" "$GAME_DIR/MONKEY1.001"
        GAME_ID="monkeycdalt"
        break
    fi
done

# Classic CD: MONKEY.000 / MONKEY.001
if [[ -z "$GAME_ID" ]]; then
    for name in MONKEY monkey; do
        if [[ -f "$INPUT_DIR/$name.000" && -f "$INPUT_DIR/$name.001" ]]; then
            cp "$INPUT_DIR/$name.000" "$GAME_DIR/MONKEY.000"
            cp "$INPUT_DIR/$name.001" "$GAME_DIR/MONKEY.001"
            GAME_ID="monkeycd"
            break
        fi
    done
fi

if [[ -z "$GAME_ID" ]]; then
    echo "ERROR: No MONKEY1.000/001 or MONKEY.000/001 found in $INPUT_DIR" >&2
    exit 1
fi

echo "Detected game variant: $GAME_ID"

GEN_ROOT="$REPO_ROOT/game/monkey1/gen"

# --- Dump CHAR blocks ---
echo ""
echo "=== Dumping CHAR blocks ==="
DUMP_DIR="$WORK/dump"
"$SCUMMRP" -g "$GAME_ID" -p "$GAME_DIR" -t CHAR -od "$DUMP_DIR"

# Locate the directory containing the CHAR blocks — differs between game variants
CHAR_DIR="$(find "$DUMP_DIR" -name "CHAR_0001" -exec dirname {} \; 2>/dev/null | head -1)"
if [[ -z "$CHAR_DIR" ]]; then
    echo "ERROR: CHAR blocks not found in scummrp output" >&2
    exit 1
fi

# --- Save raw CHAR blocks (used as templates by build_patcher.sh) ---
echo ""
echo "=== Saving CHAR blocks ==="
CHAR_OUT="$GEN_ROOT/charset/english"
mkdir -p "$CHAR_OUT"
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    src="$CHAR_DIR/$n"
    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not in dump"
        continue
    fi
    cp "$src" "$CHAR_OUT/$n"
    echo "  $n -> $CHAR_OUT/$n"
done

# --- Export BMPs (visual reference for editing Swedish glyph bitmaps) ---
echo ""
echo "=== Exporting BMPs ==="
BMP_OUT="$GEN_ROOT/charset/english_bitmaps"
mkdir -p "$BMP_OUT"
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    src="$CHAR_DIR/$n"
    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not in dump"
        continue
    fi
    "$SCUMMFONT" o "$src" "$BMP_OUT/$n.bmp"
    echo "  $n -> $BMP_OUT/$n.bmp"
done

# --- Extract dialog strings ---
echo ""
echo "=== Extracting strings ==="
STR_OUT="$GEN_ROOT/strings"
mkdir -p "$STR_OUT"
"$SCUMMTR" -g "$GAME_ID" -p "$GAME_DIR" -cwh -A aov -o -f "$STR_OUT/english.txt"
echo "  -> $STR_OUT/english.txt ($(wc -l < "$STR_OUT/english.txt") lines)"

echo ""
echo "Done."
