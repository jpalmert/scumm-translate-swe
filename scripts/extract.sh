#!/usr/bin/env bash
# extract.sh — Extract all English assets from MI1 (top-level entry point).
#
# Detects whether you have an SE PAK archive or raw classic game files and
# calls the appropriate sub-scripts. Run this once after adding your game files.
#
# Outputs (all under game/monkey1/gen/, which is gitignored):
#   game/monkey1/gen/charset/english/        — raw CHAR font blocks (used by build.sh)
#   game/monkey1/gen/charset/english_bitmaps/ — BMP visual reference for editing Swedish glyphs
#   game/monkey1/gen/strings/english.txt     — English dialog strings for translation
#
# Usage (from repo root):
#   bash scripts/extract.sh                          # PAK at default location
#   bash scripts/extract.sh /path/to/Monkey1.pak    # SE PAK file
#   bash scripts/extract.sh /path/to/game/dir/      # directory with MONKEY1.000/001
#                                                    # or MONKEY.000/001 (CD version)
#
# Default PAK location: game/monkey1/Monkey1.pak
#
# Prerequisites:
#   bash scripts/install_deps.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INPUT="${1:-$REPO_ROOT/game/monkey1/Monkey1.pak}"

if [[ -f "$INPUT" && "${INPUT,,}" == *.pak ]]; then
    echo "=== Input: SE PAK file ($INPUT) ==="
    echo ""
    bash "$REPO_ROOT/scripts/extract_pak.sh" "$INPUT"
    echo ""
    bash "$REPO_ROOT/scripts/extract_assets.sh" "$REPO_ROOT/game/monkey1"

elif [[ -d "$INPUT" ]]; then
    echo "=== Input: game directory ($INPUT) ==="
    echo ""
    bash "$REPO_ROOT/scripts/extract_assets.sh" "$INPUT"

else
    echo "ERROR: not found: $INPUT" >&2
    echo "" >&2
    echo "Usage: bash scripts/extract.sh [Monkey1.pak | game_dir]" >&2
    echo "  Monkey1.pak  — SE PAK archive (GOG or Steam)" >&2
    echo "  game_dir     — directory containing MONKEY1.000/001 or MONKEY.000/001" >&2
    echo "" >&2
    echo "Default: game/monkey1/Monkey1.pak" >&2
    exit 1
fi
