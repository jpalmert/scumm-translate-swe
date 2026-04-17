#!/usr/bin/env bash
# clean.sh — Remove build artifacts so build.sh starts fresh.
#
# The active game is determined by pwd (must be inside games/<game>/).
#
# This removes the generated .bin files and the game's dist/ binaries.
# To also remove assets extracted from the game, run: bash scripts/clean_assets.sh
#
# Usage:
#   cd games/monkey1 && bash ../../scripts/clean.sh

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
detect_game

echo "==> Removing generated charset files..."
rm -f "$REPO_ROOT/internal/charset/gen/"char_*_patched.bin
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/internal/charset/gen" 2>/dev/null || true

echo "==> Removing dist/ binaries..."
rm -f "$GAME_DIST/mi1-translate-linux"
rm -f "$GAME_DIST/mi1-translate-darwin"
rm -f "$GAME_DIST/mi1-translate-windows.exe"
rm -f "$GAME_DIST/swedish.txt"
rm -f "$GAME_DIST/"*.zip
rmdir --ignore-fail-on-non-empty "$GAME_DIST" 2>/dev/null || true

echo "Done. Run 'bash scripts/build.sh' to regenerate everything."
