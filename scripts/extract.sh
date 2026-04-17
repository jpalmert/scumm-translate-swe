#!/usr/bin/env bash
# extract.sh — Extract all English assets from a SCUMM game (top-level entry point).
#
# Detects whether you have an SE PAK archive or raw classic game files and
# calls the appropriate sub-scripts. Run this once after adding your game files.
#
# The active game is determined by the working directory (must be inside
# games/<game>/) or by passing a PAK file / game directory as argument.
#
# Outputs (all under games/<game>/gen/, which is gitignored):
#   gen/charset/english/          — raw CHAR font blocks (used by build.sh)
#   gen/charset/english_bitmaps/  — BMP visual reference for editing Swedish glyphs
#   gen/strings/english.txt       — English dialog strings for translation
#
# Usage:
#   cd games/monkey1 && bash ../../scripts/extract.sh               # PAK at default location
#   cd games/monkey1 && bash ../../scripts/extract.sh /path/to.pak  # explicit PAK path
#   bash scripts/extract.sh /path/to/game/dir/                      # explicit game dir
#
# Prerequisites:
#   bash scripts/install_deps.sh

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

INPUT="${1:-}"

if [[ -n "$INPUT" ]]; then
    # Explicit argument given
    if [[ -f "$INPUT" && "${INPUT,,}" == *.pak ]]; then
        # PAK file — still need a game context for output dirs
        detect_game "${2:-}"
        echo "=== Input: SE PAK file ($INPUT) ==="
        echo ""
        bash "$REPO_ROOT/scripts/extract_pak.sh" "$INPUT"
        echo ""
        bash "$REPO_ROOT/scripts/extract_assets.sh" "$GAME_GAME"
    elif [[ -d "$INPUT" ]]; then
        detect_game "${2:-}"
        echo "=== Input: game directory ($INPUT) ==="
        echo ""
        bash "$REPO_ROOT/scripts/extract_assets.sh" "$INPUT"
    else
        echo "ERROR: not found: $INPUT" >&2
        echo "" >&2
        echo "Usage: cd games/<game> && bash ../../scripts/extract.sh [Monkey1.pak | game_dir]" >&2
        exit 1
    fi
else
    # No argument — use default PAK location in the active game dir
    detect_game
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
fi
