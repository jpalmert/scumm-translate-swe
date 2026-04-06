#!/usr/bin/env bash
# extract_char_bitmaps.sh — Extract English CHAR blocks from the game and export as BMPs.
#
# Run this when the game is updated or when you first set up the repo on a machine
# that has the game installed. The output BMPs are committed to git under
# assets/charset/english_bitmaps/ and serve as the visual reference for editing the
# Swedish glyph bitmaps in internal/charset/bitmaps/.
#
# Pipeline:
#   Monkey1.pak  →  MONKEY1.001  →  CHAR_NNNN (scummrp)  →  CHAR_NNNN.bmp (scummfont)
#
# Reads:   game/monkey1/Monkey1.pak  (or pass a custom PAK/game-dir as $1)
# Writes:  assets/charset/english_bitmaps/CHAR_NNNN.bmp  — commit these to git
#          assets/charset/english/CHAR_NNNN              — gitignored; used by build_patcher.sh
#
# Usage (from repo root):
#   bash scripts/extract_char_bitmaps.sh
#   bash scripts/extract_char_bitmaps.sh /path/to/Monkey1.pak
#   bash scripts/extract_char_bitmaps.sh /path/to/classic/game/dir

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCUMMRP="$REPO_ROOT/bin/linux/scummrp"
SCUMMFONT="$REPO_ROOT/bin/linux/scummfont"
if [[ "$(uname)" == "Darwin" ]]; then
    SCUMMRP="$REPO_ROOT/bin/darwin/scummrp"
    SCUMMFONT="$REPO_ROOT/bin/darwin/scummfont"
fi

for bin in "$SCUMMRP" "$SCUMMFONT"; do
    if [[ ! -x "$bin" ]]; then
        echo "ERROR: $bin not found. Run bash scripts/install_deps.sh first." >&2
        exit 1
    fi
done

GAME_INPUT="${1:-$REPO_ROOT/game/monkey1/Monkey1.pak}"
OUT_DIR="$REPO_ROOT/assets/charset/english_bitmaps"
mkdir -p "$OUT_DIR"

TMPDIR_WORK="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_WORK"' EXIT

# --- Get MONKEY1.000 and MONKEY1.001 into a temp directory with uppercase names ---
if [[ -f "$GAME_INPUT" && "${GAME_INPUT,,}" == *.pak ]]; then
    echo "=== Extracting classic files from PAK ==="
    python3 "$REPO_ROOT/tools/pak.py" extract "$GAME_INPUT" "$TMPDIR_WORK/pak" 2>/dev/null
    GAME_DIR="$TMPDIR_WORK/game"
    mkdir -p "$GAME_DIR"
    cp "$TMPDIR_WORK/pak/classic/en/monkey1.000" "$GAME_DIR/MONKEY1.000"
    cp "$TMPDIR_WORK/pak/classic/en/monkey1.001" "$GAME_DIR/MONKEY1.001"
elif [[ -d "$GAME_INPUT" ]]; then
    # Caller provided a directory containing MONKEY1.000/001 (uppercase or lowercase).
    GAME_DIR="$TMPDIR_WORK/game"
    mkdir -p "$GAME_DIR"
    for f in 000 001; do
        for name in "MONKEY1.$f" "monkey1.$f"; do
            src="$GAME_INPUT/$name"
            if [[ -f "$src" ]]; then
                cp "$src" "$GAME_DIR/MONKEY1.$f"
                break
            fi
        done
        if [[ ! -f "$GAME_DIR/MONKEY1.$f" ]]; then
            echo "ERROR: MONKEY1.$f not found in $GAME_INPUT" >&2
            exit 1
        fi
    done
else
    echo "ERROR: game input not found: $GAME_INPUT" >&2
    echo "  Pass a Monkey1.pak path or a directory containing MONKEY1.000/001," >&2
    echo "  or place Monkey1.pak at game/monkey1/Monkey1.pak." >&2
    exit 1
fi

# --- Dump CHAR blocks from MONKEY1.001 ---
echo "=== Dumping CHAR blocks ==="
DUMP_DIR="$TMPDIR_WORK/dump"
"$SCUMMRP" -g monkeycdalt -p "$GAME_DIR" -t CHAR -od "$DUMP_DIR"
CHAR_DIR="$DUMP_DIR/DISK_0001/LECF/LFLF_0010"

# --- Cache raw CHAR blocks for use by build_patcher.sh ---
echo "=== Caching CHAR blocks ==="
CACHE_DIR="$REPO_ROOT/assets/charset/english"
mkdir -p "$CACHE_DIR"
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    src="$CHAR_DIR/$n"
    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not found in dump"
        continue
    fi
    cp "$src" "$CACHE_DIR/$n"
    echo "  $n -> $CACHE_DIR/$n"
done

# --- Export each CHAR block to BMP ---
echo "=== Exporting to BMP ==="
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    src="$CHAR_DIR/$n"
    bmp="$OUT_DIR/${n}.bmp"
    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not found in dump (not present in this game version)"
        continue
    fi
    "$SCUMMFONT" o "$src" "$bmp"
    echo "  $n -> $bmp"
done

echo ""
echo "Done. Review and commit assets/charset/english_bitmaps/ if the files changed."
