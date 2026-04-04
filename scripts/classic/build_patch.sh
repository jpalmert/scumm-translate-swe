#!/usr/bin/env bash
# Create BPS patch files from original vs. modified game resource files.
# The resulting .bps files are the distributable translation patches.
#
# Usage:
#   bash scripts/classic/build_patch.sh <original_dir> <modified_dir> <output_dir> [version]
#
# Arguments:
#   original_dir  Directory containing the ORIGINAL unmodified game files
#   modified_dir  Directory containing the PATCHED game files (after inject_text.sh)
#   output_dir    Where to write the .bps patch files
#   version       Version string for filenames (default: "1.0")
#
# The script diffs every file in original_dir vs. modified_dir that differs
# and creates one .bps patch per changed file.
#
# Example:
#   bash scripts/classic/build_patch.sh \
#     ~/games/monkey1_original/ \
#     ~/games/monkey1_patched/ \
#     games/monkey1/patches/ \
#     1.0

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FLIPS="$REPO_ROOT/tools/bin/flips"

if [ $# -lt 3 ]; then
    echo "Usage: $0 <original_dir> <modified_dir> <output_dir> [version]"
    exit 1
fi

ORIGINAL_DIR="$1"
MODIFIED_DIR="$2"
OUTPUT_DIR="$3"
VERSION="${4:-1.0}"

if [ ! -f "$FLIPS" ]; then
    echo "ERROR: flips not found at $FLIPS"
    echo "       Run: bash scripts/install_deps.sh"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"
PATCH_COUNT=0

for orig_file in "$ORIGINAL_DIR"/*; do
    filename="$(basename "$orig_file")"
    modified_file="$MODIFIED_DIR/$filename"

    if [ ! -f "$modified_file" ]; then
        continue
    fi

    if cmp -s "$orig_file" "$modified_file"; then
        echo "  unchanged: $filename"
        continue
    fi

    # Derive patch name: e.g. MONKEY.000 -> MONKEY_000_v1.0.bps
    patch_name="${filename//./_}_v${VERSION}.bps"
    patch_path="$OUTPUT_DIR/$patch_name"

    echo "  patching:  $filename -> $patch_name"
    "$FLIPS" --create --bps "$orig_file" "$modified_file" "$patch_path"
    PATCH_COUNT=$((PATCH_COUNT + 1))
done

echo ""
echo "Created $PATCH_COUNT patch file(s) in $OUTPUT_DIR"
echo ""
echo "To verify a patch applies correctly:"
echo "  flips --apply <patch.bps> <original_file> <output_file>"
