#!/usr/bin/env bash
# clean.sh — Remove all generated files so build_patcher.sh starts fresh
#
# Run from the repo root:
#   bash scripts/clean.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Removing dist/ binaries..."
rm -f "$REPO_ROOT/dist/classic-patcher-linux"
rm -f "$REPO_ROOT/dist/classic-patcher-darwin"
rm -f "$REPO_ROOT/dist/classic-patcher-windows.exe"
rm -f "$REPO_ROOT/dist/se-patcher-linux"
rm -f "$REPO_ROOT/dist/se-patcher-darwin"
rm -f "$REPO_ROOT/dist/se-patcher-windows.exe"
rm -f "$REPO_ROOT/dist/monkey1_swe.txt"
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/dist" 2>/dev/null || true

echo "==> Removing extracted text files..."
rm -f "$REPO_ROOT/game/monkey1/text/se_english.txt"
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/game/monkey1/text" 2>/dev/null || true

echo "Done. Run 'bash scripts/build_patcher.sh' to regenerate everything."
