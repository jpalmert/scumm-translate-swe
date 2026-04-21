#!/usr/bin/env bash
# build.sh — Build the Swedish translation patcher for the active game.
#
# The active game is determined by pwd (must be inside games/<game>/).
#
# Steps:
#   1. Verify tool binaries are present (scummtr, scummfont).
#      These are committed to git. If missing, run: bash scripts/install_deps.sh
#   2. Generate Swedish CHAR block assets (internal/charset/gen/):
#        - Use cached English CHAR blocks from games/<game>/gen/charset/english/
#          (populate with: bash scripts/extract_assets.sh)
#        - Import Swedish glyph BMPs with scummfont
#   3. Copy Swedish translation file to games/<game>/dist/
#   3b. Apply @ padding for dynamic object names (setObjectName overflow protection).
#       Uses dynamic_names.json (generated from game scripts if missing) and
#       calc_padding.py to ensure OBNA buffers are large enough for runtime replacements.
#       Only the dist copy is modified — the source swedish.txt is untouched.
#   4. Cross-compile patcher for Linux, macOS, and Windows into games/<game>/dist/
#   5. Package per-OS zip archives for distribution
#
# Output:
#   games/<game>/dist/mi1-translate-linux.zip      ← Linux release archive
#   games/<game>/dist/mi1-translate-macos.zip      ← macOS release archive
#   games/<game>/dist/mi1-translate-windows.zip    ← Windows release archive
#
# Requirements:
#   - Go 1.21+  (go build)
#   - Python 3  (calc_padding.py, find_dynamic_names.py)
#   - Tool binaries in git (scummtr, scummrp, scummfont — run install_deps.sh if missing)
#   - Extracted English CHAR blocks in games/<game>/gen/charset/english/
#     (run: cd games/<game> && bash ../../scripts/extract.sh)
#
# Usage:
#   cd games/monkey1 && bash ../../scripts/build.sh

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
detect_game "${1:-}"

ASSETS_DIR="$REPO_ROOT/internal/classic/assets"
DIST_DIR="$GAME_DIST"
TRANSLATION_SRC="$GAME_TRANSLATION/swedish.txt"

SCUMMFONT="$REPO_ROOT/bin/linux/scummfont"
if [[ "$(uname)" == "Darwin" ]]; then
    SCUMMFONT="$REPO_ROOT/bin/darwin/scummfont"
fi

mkdir -p "$ASSETS_DIR" "$DIST_DIR"

TMPDIR_BUILD="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_BUILD"' EXIT

# ---------------------------------------------------------------------------
echo "=== Step 1: Verify tool binaries ==="

missing=()
for f in \
    "$ASSETS_DIR/scummtr-linux-x64" \
    "$ASSETS_DIR/scummtr-darwin-x64" \
    "$ASSETS_DIR/scummtr-windows-x64.exe" \
    "$SCUMMFONT"
do
    [[ -f "$f" ]] || missing+=("$f")
done

if [[ ${#missing[@]} -gt 0 ]]; then
    echo "ERROR: Missing tool binaries:" >&2
    for f in "${missing[@]}"; do echo "  $f" >&2; done
    echo "" >&2
    echo "Run: bash scripts/install_deps.sh" >&2
    exit 1
fi

echo "  All tool binaries present."

# ---------------------------------------------------------------------------
echo ""
echo "=== Step 2: Generate Swedish CHAR block assets ==="

# Use committed English CHAR blocks as templates for scummfont import.
# Populate by running: bash scripts/extract_assets.sh
CHAR_CACHE="$GAME_GEN/charset/english"
missing_cache=()
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    [[ -f "$CHAR_CACHE/$n" ]] || missing_cache+=("$n")
done
if [[ ${#missing_cache[@]} -gt 0 ]]; then
    echo "ERROR: English CHAR block cache is missing files:" >&2
    for n in "${missing_cache[@]}"; do echo "  $CHAR_CACHE/$n" >&2; done
    echo "" >&2
    echo "Run: cd games/$GAME && bash ../../scripts/extract.sh" >&2
    exit 1
fi

# For each CHAR block: import the Swedish BMP into a copy of the English block.
GEN_DIR="$REPO_ROOT/internal/charset/gen"
mkdir -p "$GEN_DIR"
BITMAPS="$REPO_ROOT/internal/charset/bitmaps"

for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    lower="char_$(echo "${n#CHAR_}")_patched.bin"
    src="$CHAR_CACHE/$n"
    bmp="$BITMAPS/${n}_swedish.bmp"
    work="$TMPDIR_BUILD/work_$n"

    if [[ ! -f "$bmp" ]]; then
        echo "  SKIP $n: Swedish BMP not found"
        continue
    fi

    cp "$src" "$work"
    "$SCUMMFONT" i "$work" "$bmp"
    cp "$work" "$GEN_DIR/$lower"
    echo "  $n -> $lower ($(wc -c < "$GEN_DIR/$lower" | tr -d ' ') bytes)"
done

# ---------------------------------------------------------------------------
echo ""
echo "=== Step 3: Copy Swedish translation to dist/ ==="

if [ ! -f "$TRANSLATION_SRC" ]; then
    echo "ERROR: Translation file not found: $TRANSLATION_SRC" >&2
    exit 1
fi
cp "$TRANSLATION_SRC" "$DIST_DIR/swedish.txt"
echo "  $TRANSLATION_SRC -> $DIST_DIR/swedish.txt"

# Copy SE translation files if they exist.
for se_file in uitext_swedish.txt hints_swedish.txt; do
    src="$GAME_TRANSLATION/$se_file"
    if [ -f "$src" ]; then
        cp "$src" "$DIST_DIR/$se_file"
        echo "  $src -> $DIST_DIR/$se_file"
    fi
done

# ---------------------------------------------------------------------------
echo ""
echo "=== Step 3b: Apply @ padding for dynamic object names ==="

# Some object names are replaced at runtime by setObjectName() scripts.
# The OBNA buffer must be at least as long as the longest replacement.
# find_dynamic_names.py extracts the mapping from game scripts; calc_padding.py
# adds @ padding to the dist copy of swedish.txt where needed.

DYNNAMES_JSON="$GAME_GEN/dynamic_names.json"

if [ ! -f "$DYNNAMES_JSON" ]; then
    echo "  Generating dynamic_names.json from game scripts..."
    python3 "$REPO_ROOT/tools/find_dynamic_names.py" "$GAME_GAME" "$DYNNAMES_JSON"
fi

echo "  Checking @ padding..."
python3 "$REPO_ROOT/tools/calc_padding.py" --apply \
    --json "$DYNNAMES_JSON" \
    --translation "$DIST_DIR/swedish.txt"

# ---------------------------------------------------------------------------
echo ""
echo "=== Step 4: Cross-compile patcher ==="

GO_BIN=""
for candidate in go ~/go/bin/go /usr/local/go/bin/go; do
    if [[ "$candidate" == */* ]] && [[ -x "$candidate" ]] || [[ "$candidate" != */* ]] && command -v "$candidate" &>/dev/null; then
        GO_BIN="$candidate"
        break
    fi
done
if [ -z "$GO_BIN" ]; then
    echo "ERROR: Go not found. Install Go 1.21+ from https://go.dev/dl/" >&2
    exit 1
fi
echo "  Go: $("$GO_BIN" version | awk '{print $3}')"

cd "$REPO_ROOT"
build_binary() {
    local goos="$1" goarch="$2" out="$3"
    echo "  Building $out..."
    GOOS="$goos" GOARCH="$goarch" "$GO_BIN" build -tags buildpatcher -o "$DIST_DIR/$out" ./cmd/patcher
    echo "    -> $DIST_DIR/$out ($(du -h "$DIST_DIR/$out" | cut -f1))"
}

build_binary linux   amd64 mi1-translate-linux
build_binary darwin  amd64 mi1-translate-darwin
build_binary windows amd64 mi1-translate-windows.exe

# ---------------------------------------------------------------------------
echo ""
echo "=== Step 5: Package release archives ==="

cd "$DIST_DIR"

# Collect all translation files present in dist/
TRANS_FILES=(swedish.txt)
for se_file in uitext_swedish.txt hints_swedish.txt; do
    [ -f "$se_file" ] && TRANS_FILES+=("$se_file")
done

zip -j mi1-translate-linux.zip   mi1-translate-linux     "${TRANS_FILES[@]}"
zip -j mi1-translate-macos.zip   mi1-translate-darwin     "${TRANS_FILES[@]}"
zip -j mi1-translate-windows.zip mi1-translate-windows.exe "${TRANS_FILES[@]}"

rm -f mi1-translate-linux mi1-translate-darwin mi1-translate-windows.exe "${TRANS_FILES[@]}"
cd "$REPO_ROOT"

echo "  Created release archives."

# ---------------------------------------------------------------------------
echo ""
echo "=== Done! ==="
echo ""
echo "Output in $DIST_DIR/:"
ls -lh "$DIST_DIR/"
echo ""
echo "Upload the .zip files to a GitHub release:"
echo "  gh release create v1.0.0 $DIST_DIR/*.zip --title 'v1.0.0'"
