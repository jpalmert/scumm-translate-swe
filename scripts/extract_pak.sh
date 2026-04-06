#!/usr/bin/env bash
# extract_pak.sh — Unpack MONKEY1.000/001 from a MI1 Special Edition PAK archive.
#
# The PAK file bundles the classic SCUMM data files used in the "classic mode"
# of MI1SE. This script pulls them out so extract_assets.sh can work on them.
#
# Reads:   Monkey1.pak (default: game/monkey1/Monkey1.pak)
# Writes:  game/monkey1/MONKEY1.000
#          game/monkey1/MONKEY1.001
#
# Usage (from repo root):
#   bash scripts/extract_pak.sh
#   bash scripts/extract_pak.sh /path/to/Monkey1.pak

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PAKPY="$REPO_ROOT/tools/pak.py"
PAK="${1:-$REPO_ROOT/game/monkey1/Monkey1.pak}"

if [[ ! -f "$PAK" ]]; then
    echo "ERROR: PAK file not found: $PAK" >&2
    echo "  Pass the path to Monkey1.pak, or place it at game/monkey1/Monkey1.pak" >&2
    exit 1
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "=== Extracting PAK ==="
python3 "$PAKPY" extract "$PAK" "$WORK/pak"

OUT_DIR="$REPO_ROOT/game/monkey1"
mkdir -p "$OUT_DIR"
cp "$WORK/pak/classic/en/monkey1.000" "$OUT_DIR/MONKEY1.000"
cp "$WORK/pak/classic/en/monkey1.001" "$OUT_DIR/MONKEY1.001"

echo "  -> $OUT_DIR/MONKEY1.000"
echo "  -> $OUT_DIR/MONKEY1.001"
echo "Done. Run: bash scripts/extract_assets.sh $OUT_DIR"
