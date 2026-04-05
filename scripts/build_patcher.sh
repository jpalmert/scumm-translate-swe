#!/usr/bin/env bash
# build_patcher.sh — Build the MI1 Swedish translation patchers
#
# Run from the repo root:
#   bash scripts/build_patcher.sh
#
# What this does:
#   1. Downloads scummtr binaries into internal/classic/assets/
#      (skipped if already present — these are committed to git after first run)
#   2. Cross-compiles both patchers for Linux, macOS, and Windows
#   3. Places the output binaries and the loose translation file in dist/:
#        dist/classic-patcher-linux
#        dist/classic-patcher-darwin
#        dist/classic-patcher-windows.exe
#        dist/se-patcher-linux
#        dist/se-patcher-darwin
#        dist/se-patcher-windows.exe
#        dist/monkey1_swe.txt     ← ship this alongside the binaries
#
# Requirements:
#   - Go 1.21+  (go build)
#   - curl      (for downloading scummtr if not already in assets/)
#   - unzip     (for extracting the Windows scummtr zip)
#
# The resulting dist/ binaries embed the scummtr tool internally. The only
# loose file is monkey1_swe.txt — users can edit it before running the patcher.
#
# Usage of the built patchers (for users):
#   classic-patcher-linux  /path/to/game/dir  [monkey1_swe.txt]
#   se-patcher-linux       /path/to/Monkey1.pak [output.pak] [monkey1_swe.txt]
# After patching, set the in-game language to French.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ASSETS_DIR="$REPO_ROOT/internal/classic/assets"
DIST_DIR="$REPO_ROOT/dist"
TRANSLATION_SRC="$REPO_ROOT/translation/monkey1/monkey1_swe.txt"

SCUMMTR_VERSION="0.5.1"
SCUMMTR_BASE_URL="https://github.com/dwatteau/scummtr/releases/download/v${SCUMMTR_VERSION}"

mkdir -p "$ASSETS_DIR" "$DIST_DIR"

# Check for unzip upfront — needed for macOS and Windows scummtr downloads.
if ! command -v unzip &>/dev/null; then
    echo "ERROR: unzip not found. Install with: sudo apt-get install unzip"
    exit 1
fi

TMPDIR_DL="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_DL"' EXIT

echo "=== Step 1: Prepare scummtr binaries ==="

# --- Linux x64 ---
if [ -f "$ASSETS_DIR/scummtr-linux-x64" ]; then
    echo "  Linux binary already present: $ASSETS_DIR/scummtr-linux-x64"
else
    echo "  Downloading scummtr v${SCUMMTR_VERSION} for Linux..."
    curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-linux86.tar.gz" \
        | tar xz -C "$TMPDIR_DL"
    cp "$TMPDIR_DL/scummtr-${SCUMMTR_VERSION}-linux86/linux-x64/scummtr" \
        "$ASSETS_DIR/scummtr-linux-x64"
    chmod +x "$ASSETS_DIR/scummtr-linux-x64"
    echo "  Installed: $ASSETS_DIR/scummtr-linux-x64"
fi

# --- macOS (Darwin) x64 ---
if [ -f "$ASSETS_DIR/scummtr-darwin-x64" ]; then
    echo "  macOS binary already present: $ASSETS_DIR/scummtr-darwin-x64"
else
    echo "  Downloading scummtr v${SCUMMTR_VERSION} for macOS..."
    curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-macos.zip" \
        -o "$TMPDIR_DL/mac.zip"
    unzip -q "$TMPDIR_DL/mac.zip" -d "$TMPDIR_DL/mac"
    MACOS_BIN="$(find "$TMPDIR_DL/mac" -name "scummtr" | head -1)"
    if [ -z "$MACOS_BIN" ]; then
        echo "ERROR: Could not find scummtr binary in macOS zip."
        echo "       Check the release at: ${SCUMMTR_BASE_URL}/"
        exit 1
    fi
    cp "$MACOS_BIN" "$ASSETS_DIR/scummtr-darwin-x64"
    chmod +x "$ASSETS_DIR/scummtr-darwin-x64"
    echo "  Installed: $ASSETS_DIR/scummtr-darwin-x64"
fi

# --- Windows x64 ---
if [ -f "$ASSETS_DIR/scummtr-windows-x64.exe" ]; then
    echo "  Windows binary already present: $ASSETS_DIR/scummtr-windows-x64.exe"
else
    echo "  Downloading scummtr v${SCUMMTR_VERSION} for Windows..."
    curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-win32.zip" \
        -o "$TMPDIR_DL/win32.zip"
    unzip -q "$TMPDIR_DL/win32.zip" -d "$TMPDIR_DL/win32"
    if [ -f "$TMPDIR_DL/win32/scummtr-${SCUMMTR_VERSION}-win32/scummtr.exe" ]; then
        cp "$TMPDIR_DL/win32/scummtr-${SCUMMTR_VERSION}-win32/scummtr.exe" \
            "$ASSETS_DIR/scummtr-windows-x64.exe"
    else
        echo "ERROR: Could not find scummtr.exe in Windows zip."
        echo "       Check the release at: ${SCUMMTR_BASE_URL}/"
        echo "       Then update the extraction path in this script."
        exit 1
    fi
    echo "  Installed: $ASSETS_DIR/scummtr-windows-x64.exe"
fi

echo ""
echo "=== Step 2: Copy Swedish translation to dist/ ==="

if [ ! -f "$TRANSLATION_SRC" ]; then
    echo "ERROR: Translation file not found: $TRANSLATION_SRC"
    echo "       Copy monkey1_swe.txt from the monkeycd_swe repo to translation/monkey1/"
    exit 1
fi

cp "$TRANSLATION_SRC" "$DIST_DIR/monkey1_swe.txt"
echo "  Copied: $TRANSLATION_SRC"
echo "        → $DIST_DIR/monkey1_swe.txt"

echo ""
echo "=== Step 3: Cross-compile patchers ==="

# Verify Go is available (try common install locations)
GO_BIN=""
for candidate in go ~/go/bin/go /usr/local/go/bin/go; do
    if command -v "$candidate" &>/dev/null 2>&1; then
        GO_BIN="$candidate"
        break
    fi
done
if [ -z "$GO_BIN" ]; then
    echo "ERROR: Go not found. Install Go 1.21+ from https://go.dev/dl/"
    echo "       Then ensure 'go' is on your PATH (or add ~/go/bin to PATH)."
    exit 1
fi

GO_VERSION="$("$GO_BIN" version | awk '{print $3}')"
echo "  Go: $GO_VERSION"

cd "$REPO_ROOT"

build_binary() {
    local goos="$1"
    local goarch="$2"
    local cmd="$3"    # cmd/classic-patcher or cmd/se-patcher
    local out="$4"
    echo "  Building $out..."
    GOOS="$goos" GOARCH="$goarch" "$GO_BIN" build -o "$DIST_DIR/$out" "./$cmd"
    echo "    → $DIST_DIR/$out ($(du -h "$DIST_DIR/$out" | cut -f1))"
}

# classic-patcher
build_binary linux   amd64 cmd/classic-patcher classic-patcher-linux
build_binary darwin  amd64 cmd/classic-patcher classic-patcher-darwin
build_binary windows amd64 cmd/classic-patcher classic-patcher-windows.exe

# se-patcher
build_binary linux   amd64 cmd/se-patcher se-patcher-linux
build_binary darwin  amd64 cmd/se-patcher se-patcher-darwin
build_binary windows amd64 cmd/se-patcher se-patcher-windows.exe

echo ""
echo "=== Done! ==="
echo ""
echo "Output in dist/:"
ls -lh "$DIST_DIR/"
echo ""
echo "Distribute all files in dist/ together (binaries + monkey1_swe.txt)."
echo ""
echo "Usage:"
echo "  Classic (ScummVM):  classic-patcher-linux  /path/to/game/dir"
echo "  SE (GOG/Steam):     se-patcher-linux        /path/to/Monkey1.pak"
echo ""
echo "After patching, set the in-game language to French."
