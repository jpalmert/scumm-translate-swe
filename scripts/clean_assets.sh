#!/usr/bin/env bash
# clean_assets.sh — Remove all assets extracted from game files.
#
# The active game is determined by pwd (must be inside games/<game>/).
#
# Undoes scripts/extract.sh. After running this, re-run extract.sh to
# regenerate everything.
#
# Removes:
#   games/<game>/gen/              — all extracted assets (CHAR blocks, BMPs, strings)
#   games/<game>/game/MONKEY1.000/001 — only if Monkey1.pak is present (i.e. they were
#                                       unpacked from the PAK rather than provided directly)
#
# Does NOT remove:
#   internal/charset/bitmaps/      — Swedish glyph BMPs (hand-edited source files)
#   internal/charset/gen/          — use clean.sh to remove build artifacts

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
detect_game

echo "==> Removing extracted assets..."
rm -rf "$GAME_GEN"

# Only remove the unpacked classic files if they were extracted from a PAK.
# If no PAK is present the user placed these files directly — leave them alone.
if [[ -f "$GAME_GAME/Monkey1.pak" ]]; then
    echo "==> Removing classic files unpacked from PAK..."
    rm -f "$GAME_GAME/MONKEY1.000"
    rm -f "$GAME_GAME/MONKEY1.001"
fi

echo "Done. Run 'bash scripts/extract.sh' to regenerate."
