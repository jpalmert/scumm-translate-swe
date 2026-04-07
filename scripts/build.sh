#!/usr/bin/env bash
# build.sh — Build the MI1 Swedish translation patcher
#
# Run from the repo root:
#   bash scripts/build.sh
#
# Steps:
#   1. Verify tool binaries are present (scummtr, scummfont).
#      These are committed to git. If missing, run: bash scripts/install_deps.sh
#   2. Generate Swedish CHAR block assets (internal/charset/gen/):
#        - Use cached English CHAR blocks from game/monkey1/gen/charset/english/
#          (populate with: bash scripts/extract_assets.sh)
#        - Import Swedish glyph BMPs with scummfont
#   3. Copy Swedish translation file to dist/
#   4. Cross-compile patcher for Linux, macOS, and Windows into dist/
#
# Output:
#   dist/mi1-translate-linux
#   dist/mi1-translate-darwin
#   dist/mi1-translate-windows.exe
#   dist/swedish.txt     ← ship alongside the binaries
#
# Requirements:
#   - Go 1.21+  (go build)
#   - Tool binaries in git (scummtr, scummrp, scummfont — run install_deps.sh if missing)
#   - Extracted English CHAR blocks in game/monkey1/gen/charset/english/
#     (run: bash scripts/extract.sh [Monkey1.pak | game_dir])
#
# Usage of the built patcher (for users):
#   Place mi1-translate-linux and swedish.txt next to your game files and run it.
#   After patching, start a new game to see Swedish text.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ASSETS_DIR="$REPO_ROOT/internal/classic/assets"
DIST_DIR="$REPO_ROOT/dist"
TRANSLATION_SRC="$REPO_ROOT/translation/monkey1/swedish.txt"

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
CHAR_CACHE="$REPO_ROOT/game/monkey1/gen/charset/english"
missing_cache=()
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    [[ -f "$CHAR_CACHE/$n" ]] || missing_cache+=("$n")
done
if [[ ${#missing_cache[@]} -gt 0 ]]; then
    echo "ERROR: English CHAR block cache is missing files:" >&2
    for n in "${missing_cache[@]}"; do echo "  $CHAR_CACHE/$n" >&2; done
    echo "" >&2
    echo "Run: bash scripts/extract.sh [Monkey1.pak | game_dir]" >&2
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
    echo "ERROR: Translation file not found: $TRANSLATION_SRC"
    exit 1
fi
cp "$TRANSLATION_SRC" "$DIST_DIR/swedish.txt"
echo "  $TRANSLATION_SRC -> dist/swedish.txt"

# ---------------------------------------------------------------------------
echo ""
echo "=== Step 4: Cross-compile patcher ==="

GO_BIN=""
for candidate in go ~/go/bin/go /usr/local/go/bin/go; do
    if command -v "$candidate" &>/dev/null 2>&1; then
        GO_BIN="$candidate"
        break
    fi
done
if [ -z "$GO_BIN" ]; then
    echo "ERROR: Go not found. Install Go 1.21+ from https://go.dev/dl/"
    exit 1
fi
echo "  Go: $("$GO_BIN" version | awk '{print $3}')"

cd "$REPO_ROOT"
build_binary() {
    local goos="$1" goarch="$2" out="$3"
    echo "  Building $out..."
    GOOS="$goos" GOARCH="$goarch" "$GO_BIN" build -o "$DIST_DIR/$out" ./cmd/patcher
    echo "    -> $DIST_DIR/$out ($(du -h "$DIST_DIR/$out" | cut -f1))"
}

build_binary linux   amd64 mi1-translate-linux
build_binary darwin  amd64 mi1-translate-darwin
build_binary windows amd64 mi1-translate-windows.exe

# ---------------------------------------------------------------------------
echo ""
echo "=== Done! ==="
echo ""
echo "Output in dist/:"
ls -lh "$DIST_DIR/"
echo ""
echo "Distribute all files in dist/ together (binaries + swedish.txt)."
echo "After patching, start a new game to see Swedish text."
