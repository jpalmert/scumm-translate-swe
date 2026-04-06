#!/usr/bin/env bash
# clean.sh — Remove all generated files so build_patcher.sh starts fresh
#
# Run from the repo root:
#   bash scripts/clean.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Removing generated charset files..."
rm -f "$REPO_ROOT/internal/charset/gen/"char_*_patched.bin
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/internal/charset/gen" 2>/dev/null || true
rm -rf "$REPO_ROOT/assets/charset/english"

echo "==> Removing dist/ binaries..."
rm -f "$REPO_ROOT/dist/mi1-translate-linux"
rm -f "$REPO_ROOT/dist/mi1-translate-darwin"
rm -f "$REPO_ROOT/dist/mi1-translate-windows.exe"
rm -f "$REPO_ROOT/dist/monkey1.txt"
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/dist" 2>/dev/null || true

echo "==> Removing extracted text files..."
rm -f "$REPO_ROOT/assets/strings/english.txt"
rmdir --ignore-fail-on-non-empty "$REPO_ROOT/assets/strings" 2>/dev/null || true

echo "Done. Run 'bash scripts/build_patcher.sh' to regenerate everything."
