#!/usr/bin/env bash
# clean_assets.sh — Remove all assets extracted from game files.
#
# Undoes scripts/extract.sh. After running this, re-run extract.sh to
# regenerate everything.
#
# Removes:
#   game/monkey1/gen/              — all extracted assets (CHAR blocks, BMPs, strings)
#   game/monkey1/MONKEY1.000/001   — only if Monkey1.pak is present (i.e. they were
#                                    unpacked from the PAK rather than provided directly)
#
# Does NOT remove:
#   internal/charset/bitmaps/      — Swedish glyph BMPs (hand-edited source files)
#   internal/charset/gen/          — use clean.sh to remove build artifacts

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Removing extracted assets..."
rm -rf "$REPO_ROOT/game/monkey1/gen"

echo "==> Removing generated dynamic name mapping..."
rm -f "$REPO_ROOT/translation/monkey1/dynamic_names.json"

# Only remove the unpacked classic files if they were extracted from a PAK.
# If no PAK is present the user placed these files directly — leave them alone.
if [[ -f "$REPO_ROOT/game/monkey1/Monkey1.pak" ]]; then
    echo "==> Removing classic files unpacked from PAK..."
    rm -f "$REPO_ROOT/game/monkey1/MONKEY1.000"
    rm -f "$REPO_ROOT/game/monkey1/MONKEY1.001"
fi

rmdir --ignore-fail-on-non-empty "$REPO_ROOT/game/monkey1" 2>/dev/null || true
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/game" 2>/dev/null || true

echo "Done. Run 'bash scripts/extract.sh' to regenerate."
