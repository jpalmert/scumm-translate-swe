#!/usr/bin/env bash
# extract_pak.sh — Unpack MONKEY1.000/001 from an SE PAK archive.
#
# The PAK file bundles the classic SCUMM data files used in "classic mode".
# This script pulls them out so extract_assets.sh can work on them.
#
# The active game is determined by pwd (must be inside games/<game>/).
#
# Reads:   Monkey1.pak (default: games/<game>/game/Monkey1.pak)
# Writes:  games/<game>/game/MONKEY1.000
#          games/<game>/game/MONKEY1.001
#
# Usage:
#   cd games/monkey1 && bash ../../scripts/extract_pak.sh
#   cd games/monkey1 && bash ../../scripts/extract_pak.sh /path/to/Monkey1.pak

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
detect_game "${1:-}"

PAKPY="$REPO_ROOT/tools/pak.py"
PAK="${1:-$GAME_GAME/Monkey1.pak}"

if [[ ! -f "$PAK" ]]; then
    echo "ERROR: PAK file not found: $PAK" >&2
    echo "  Pass the path to Monkey1.pak, or place it at $GAME_GAME/Monkey1.pak" >&2
    exit 1
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "=== Extracting PAK ==="
python3 "$PAKPY" extract "$PAK" "$WORK/pak"

mkdir -p "$GAME_GAME"
cp "$WORK/pak/classic/en/monkey1.000" "$GAME_GAME/MONKEY1.000"
cp "$WORK/pak/classic/en/monkey1.001" "$GAME_GAME/MONKEY1.001"

echo "  -> $GAME_GAME/MONKEY1.000"
echo "  -> $GAME_GAME/MONKEY1.001"
echo "Done."
