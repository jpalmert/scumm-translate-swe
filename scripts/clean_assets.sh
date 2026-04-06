#!/usr/bin/env bash
# clean_assets.sh — Remove all assets extracted from game files.
#
# Undoes scripts/extract.sh. After running this, re-run extract.sh to
# regenerate everything.
#
# Removes:
#   assets/charset/english/        — raw CHAR font blocks
#   assets/charset/english_bitmaps/ — English reference BMPs
#   assets/strings/                — extracted dialog strings
#   game/monkey1/MONKEY1.000       — classic files unpacked from PAK (if present)
#   game/monkey1/MONKEY1.001
#
# Does NOT remove:
#   internal/charset/bitmaps/      — Swedish glyph BMPs (hand-edited source files)
#   internal/charset/gen/          — use clean.sh to remove build artifacts

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Removing extracted charset assets..."
rm -rf "$REPO_ROOT/assets/charset/english"
rm -rf "$REPO_ROOT/assets/charset/english_bitmaps"

echo "==> Removing extracted strings..."
rm -rf "$REPO_ROOT/assets/strings"

echo "==> Removing unpacked classic game files..."
rm -f "$REPO_ROOT/game/monkey1/MONKEY1.000"
rm -f "$REPO_ROOT/game/monkey1/MONKEY1.001"
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/game/monkey1" 2>/dev/null || true
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/game" 2>/dev/null || true

echo "Done. Run 'bash scripts/extract.sh' to regenerate."
