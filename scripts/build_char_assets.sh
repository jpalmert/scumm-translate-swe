#!/usr/bin/env bash
# build_char_assets.sh — Produce the Swedish CHAR block binaries embedded in the patcher.
#
# This script is part of the dist build pipeline. Run it (or let build_patcher.sh
# call it) before `go build` so the //go:embed directives in charset.go can find
# the generated .bin files in internal/charset/gen/.
#
# How it works
# ------------
# For each classic CHAR block (0001, 0002, 0003, 0004, 0006):
#   1. Extract the original English CHAR block from the game using scummrp.
#      (MONKEY1.001 is extracted from Monkey1.pak if a PAK file is given.)
#   2. Import the Swedish glyph BMP (internal/charset/bitmaps/CHAR_NNNN_swedish.bmp)
#      into a copy of the English block using `scummfont i`. scummfont replaces glyph
#      pixel data in-place and updates the CHAR block's internal size fields.
#   3. Write the result to internal/charset/gen/char_NNNN_patched.bin, which is
#      then embedded by Go at compile time.
#
# Source files (committed to git):
#   internal/charset/bitmaps/CHAR_NNNN_swedish.bmp  — Swedish glyph bitmaps
#   assets/charset/english_bitmaps/CHAR_NNNN.bmp    — English reference BMPs
#     (visual reference only; the build uses the live CHAR blocks from the game)
#
# Generated files (gitignored, produced by this script):
#   internal/charset/gen/char_NNNN_patched.bin
#
# To update the Swedish glyphs:
#   1. Edit the relevant BMP in internal/charset/bitmaps/
#   2. Run this script (with the game available)
#   3. Commit the BMP; the .bin is regenerated automatically on next build
#
# To update english_bitmaps/ after a game update:
#   Run bash scripts/extract_char_bitmaps.sh and commit the changed BMPs.
#
# Usage (from repo root):
#   bash scripts/build_char_assets.sh
#   bash scripts/build_char_assets.sh /path/to/Monkey1.pak
#   bash scripts/build_char_assets.sh /path/to/classic/game/dir

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
BITMAPS="$REPO_ROOT/internal/charset/bitmaps"
GEN="$REPO_ROOT/internal/charset/gen"
mkdir -p "$GEN"

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
echo "=== Dumping CHAR blocks from game ==="
DUMP_DIR="$TMPDIR_WORK/dump"
"$SCUMMRP" -g monkeycdalt -p "$GAME_DIR" -t CHAR -od "$DUMP_DIR"
CHAR_DIR="$DUMP_DIR/DISK_0001/LECF/LFLF_0010"

# --- Import Swedish BMPs and write patched .bin files ---
echo "=== Building charset assets ==="
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    lower="char_$(echo "${n#CHAR_}")_patched.bin"
    bmp="$BITMAPS/${n}_swedish.bmp"
    src="$CHAR_DIR/$n"
    work="$TMPDIR_WORK/work_$n"
    out="$GEN/$lower"

    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not found in game dump"
        continue
    fi
    if [[ ! -f "$bmp" ]]; then
        echo "  SKIP $n: Swedish BMP not found at $bmp"
        continue
    fi

    cp "$src" "$work"
    "$SCUMMFONT" i "$work" "$bmp"
    cp "$work" "$out"
    echo "  $n -> $lower ($(wc -c < "$out" | tr -d ' ') bytes)"
done

echo ""
echo "Done. Run 'go build ./cmd/patcher' to produce the patcher binary."
