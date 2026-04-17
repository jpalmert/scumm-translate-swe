#!/usr/bin/env bash
# extract.sh — Extract all English assets from a SCUMM game (top-level entry point).
#
# Detects whether games/<game>/game/ contains an SE PAK archive or raw classic
# game files and calls the appropriate sub-scripts. Run once after placing your
# game files in games/<game>/game/.
#
# Outputs (all under games/<game>/gen/, which is gitignored):
#   gen/charset/english/          — raw CHAR font blocks (used by build.sh)
#   gen/charset/english_bitmaps/  — BMP visual reference for editing Swedish glyphs
#   gen/strings/english.txt       — English dialog strings for translation
#
# Usage:
#   bash scripts/extract.sh monkey1                     # game name
#   cd games/monkey1 && bash ../../scripts/extract.sh   # detect from pwd
#
# Prerequisites:
#   bash scripts/install_deps.sh

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
detect_game "${1:-}"

DEFAULT_PAK="$GAME_GAME/Monkey1.pak"
if [[ -f "$DEFAULT_PAK" ]]; then
    echo "=== Input: SE PAK file ($DEFAULT_PAK) ==="
    echo ""
    bash "$REPO_ROOT/scripts/extract_pak.sh" "$DEFAULT_PAK"
    echo ""
    bash "$REPO_ROOT/scripts/extract_assets.sh" "$GAME_GAME"
elif [[ -d "$GAME_GAME" ]]; then
    echo "=== Input: game directory ($GAME_GAME) ==="
    echo ""
    bash "$REPO_ROOT/scripts/extract_assets.sh" "$GAME_GAME"
else
    echo "ERROR: No game files found." >&2
    echo "  Place Monkey1.pak (or MONKEY1.000/001) in $GAME_GAME/" >&2
    exit 1
fi
