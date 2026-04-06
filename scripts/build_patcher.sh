#!/usr/bin/env bash
# build_patcher.sh — Build the MI1 Swedish translation patcher
#
# Run from the repo root:
#   bash scripts/build_patcher.sh [Monkey1.pak | game_dir]
#
# Steps:
#   1. Prepare scummtr binaries (downloaded once into internal/classic/assets/;
#      committed to git so subsequent builds skip this step)
#   2. Generate Swedish CHAR block assets (internal/charset/gen/):
#        - Extract MONKEY1.001 from Monkey1.pak (or use provided game dir)
#        - Dump CHAR blocks with scummrp
#        - Import Swedish glyph BMPs with scummfont
#   3. Copy Swedish translation file to dist/
#   4. Cross-compile patcher for Linux, macOS, and Windows into dist/
#
# Output:
#   dist/mi1-translate-linux
#   dist/mi1-translate-darwin
#   dist/mi1-translate-windows.exe
#   dist/monkey1.txt     ← ship alongside the binaries
#
# Requirements:
#   - Go 1.21+  (go build)
#   - curl, unzip  (for scummtr download if not already cached)
#   - Monkey1.pak or MONKEY1.000/001 (for charset asset generation)
#     Default location: game/monkey1/Monkey1.pak
#
# Usage of the built patcher (for users):
#   Place mi1-translate-linux and monkey1.txt next to your game files and run it.
#   After patching, start a new game to see Swedish text.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ASSETS_DIR="$REPO_ROOT/internal/classic/assets"
DIST_DIR="$REPO_ROOT/dist"
TRANSLATION_SRC="$REPO_ROOT/translation/monkey1/monkey1.txt"
GAME_INPUT="${1:-$REPO_ROOT/game/monkey1/Monkey1.pak}"

SCUMMTR_VERSION="0.5.1"
SCUMMTR_BASE_URL="https://github.com/dwatteau/scummtr/releases/download/v${SCUMMTR_VERSION}"

SCUMMRP="$REPO_ROOT/bin/linux/scummrp"
SCUMMFONT="$REPO_ROOT/bin/linux/scummfont"
if [[ "$(uname)" == "Darwin" ]]; then
    SCUMMRP="$REPO_ROOT/bin/darwin/scummrp"
    SCUMMFONT="$REPO_ROOT/bin/darwin/scummfont"
fi

mkdir -p "$ASSETS_DIR" "$DIST_DIR"

if ! command -v unzip &>/dev/null; then
    echo "ERROR: unzip not found. Install with: sudo apt-get install unzip"
    exit 1
fi

TMPDIR_BUILD="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_BUILD"' EXIT

# ---------------------------------------------------------------------------
echo "=== Step 1: Prepare scummtr binaries ==="

if [ -f "$ASSETS_DIR/scummtr-linux-x64" ]; then
    echo "  Linux binary already present"
else
    echo "  Downloading scummtr v${SCUMMTR_VERSION} for Linux..."
    curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-linux86.tar.gz" \
        | tar xz -C "$TMPDIR_BUILD"
    cp "$TMPDIR_BUILD/scummtr-${SCUMMTR_VERSION}-linux86/linux-x64/scummtr" \
        "$ASSETS_DIR/scummtr-linux-x64"
    chmod +x "$ASSETS_DIR/scummtr-linux-x64"
    echo "  Installed: $ASSETS_DIR/scummtr-linux-x64"
fi

if [ -f "$ASSETS_DIR/scummtr-darwin-x64" ]; then
    echo "  macOS binary already present"
else
    echo "  Downloading scummtr v${SCUMMTR_VERSION} for macOS..."
    curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-macos.zip" \
        -o "$TMPDIR_BUILD/mac.zip"
    unzip -q "$TMPDIR_BUILD/mac.zip" -d "$TMPDIR_BUILD/mac"
    MACOS_BIN="$(find "$TMPDIR_BUILD/mac" -name "scummtr" | head -1)"
    if [ -z "$MACOS_BIN" ]; then
        echo "ERROR: Could not find scummtr binary in macOS zip."
        exit 1
    fi
    cp "$MACOS_BIN" "$ASSETS_DIR/scummtr-darwin-x64"
    chmod +x "$ASSETS_DIR/scummtr-darwin-x64"
    echo "  Installed: $ASSETS_DIR/scummtr-darwin-x64"
fi

if [ -f "$ASSETS_DIR/scummtr-windows-x64.exe" ]; then
    echo "  Windows binary already present"
else
    echo "  Downloading scummtr v${SCUMMTR_VERSION} for Windows..."
    curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-win32.zip" \
        -o "$TMPDIR_BUILD/win32.zip"
    unzip -q "$TMPDIR_BUILD/win32.zip" -d "$TMPDIR_BUILD/win32"
    if [ ! -f "$TMPDIR_BUILD/win32/scummtr-${SCUMMTR_VERSION}-win32/scummtr.exe" ]; then
        echo "ERROR: Could not find scummtr.exe in Windows zip."
        exit 1
    fi
    cp "$TMPDIR_BUILD/win32/scummtr-${SCUMMTR_VERSION}-win32/scummtr.exe" \
        "$ASSETS_DIR/scummtr-windows-x64.exe"
    echo "  Installed: $ASSETS_DIR/scummtr-windows-x64.exe"
fi

# ---------------------------------------------------------------------------
echo ""
echo "=== Step 2: Generate Swedish CHAR block assets ==="

for bin in "$SCUMMRP" "$SCUMMFONT"; do
    if [[ ! -x "$bin" ]]; then
        echo "ERROR: $bin not found. Run bash scripts/install_deps.sh first." >&2
        exit 1
    fi
done

# Extract MONKEY1.000/001 from PAK (or use a provided game directory).
if [[ -f "$GAME_INPUT" && "${GAME_INPUT,,}" == *.pak ]]; then
    echo "  Extracting classic files from $GAME_INPUT..."
    python3 "$REPO_ROOT/tools/pak.py" extract "$GAME_INPUT" "$TMPDIR_BUILD/pak" 2>/dev/null
    GAME_DIR="$TMPDIR_BUILD/game"
    mkdir -p "$GAME_DIR"
    cp "$TMPDIR_BUILD/pak/classic/en/monkey1.000" "$GAME_DIR/MONKEY1.000"
    cp "$TMPDIR_BUILD/pak/classic/en/monkey1.001" "$GAME_DIR/MONKEY1.001"
elif [[ -d "$GAME_INPUT" ]]; then
    GAME_DIR="$TMPDIR_BUILD/game"
    mkdir -p "$GAME_DIR"
    for f in 000 001; do
        for name in "MONKEY1.$f" "monkey1.$f"; do
            if [[ -f "$GAME_INPUT/$name" ]]; then
                cp "$GAME_INPUT/$name" "$GAME_DIR/MONKEY1.$f"
                break
            fi
        done
        if [[ ! -f "$GAME_DIR/MONKEY1.$f" ]]; then
            echo "ERROR: MONKEY1.$f not found in $GAME_INPUT" >&2
            exit 1
        fi
    done
else
    echo "ERROR: game input not found: $GAME_INPUT" >&2
    echo "  Usage: bash scripts/build_patcher.sh [Monkey1.pak | game_dir]" >&2
    echo "  Default location: game/monkey1/Monkey1.pak" >&2
    exit 1
fi

# Dump CHAR blocks from MONKEY1.001 using scummrp.
DUMP_DIR="$TMPDIR_BUILD/char_dump"
"$SCUMMRP" -g monkeycdalt -p "$GAME_DIR" -t CHAR -od "$DUMP_DIR"
CHAR_DIR="$DUMP_DIR/DISK_0001/LECF/LFLF_0010"

# For each CHAR block: import the Swedish BMP and write the patched .bin.
GEN_DIR="$REPO_ROOT/internal/charset/gen"
mkdir -p "$GEN_DIR"
BITMAPS="$REPO_ROOT/internal/charset/bitmaps"

for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    lower="char_$(echo "${n#CHAR_}")_patched.bin"
    src="$CHAR_DIR/$n"
    bmp="$BITMAPS/${n}_swedish.bmp"
    work="$TMPDIR_BUILD/work_$n"

    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not found in game dump"
        continue
    fi
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
cp "$TRANSLATION_SRC" "$DIST_DIR/monkey1.txt"
echo "  $TRANSLATION_SRC -> dist/monkey1.txt"

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
echo "Distribute all files in dist/ together (binaries + monkey1.txt)."
echo "After patching, start a new game to see Swedish text."
