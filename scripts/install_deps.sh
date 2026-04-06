#!/usr/bin/env bash
# install_deps.sh — Download scummtr tool binaries for all platforms
#
# Populates:
#   bin/linux/          — developer tools for Linux
#   bin/darwin/         — developer tools for macOS
#   internal/classic/assets/scummtr-{linux,darwin,windows}-x64[.exe]
#   internal/charset/assets/scummrp-{linux,darwin,windows}-x64[.exe]
#
# Both bin/ and internal/*/assets/ are committed to the repo so devs don't
# need to run this script normally. Run it only when upgrading scummtr or
# if the committed binaries are missing or corrupted.
#
# Usage (from repo root):
#   bash scripts/install_deps.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCUMMTR_VERSION="0.5.1"
BASE_URL="https://github.com/dwatteau/scummtr/releases/download/v${SCUMMTR_VERSION}"

TMPDIR_DL="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_DL"' EXIT

echo "=== Downloading scummtr v${SCUMMTR_VERSION} for all platforms ==="

# --- Linux ---
echo ""
echo "  [linux] Downloading..."
curl -sL "${BASE_URL}/scummtr-${SCUMMTR_VERSION}-linux86.tar.gz" \
    | tar xz -C "$TMPDIR_DL"
LINUX_SRC="$TMPDIR_DL/scummtr-${SCUMMTR_VERSION}-linux86/linux-x64"

mkdir -p "$REPO_ROOT/bin/linux"
for tool in scummtr scummrp scummfont FontXY; do
    cp "$LINUX_SRC/$tool" "$REPO_ROOT/bin/linux/$tool"
    chmod +x "$REPO_ROOT/bin/linux/$tool"
done
echo "  [linux] bin/linux/ updated"

cp "$LINUX_SRC/scummtr" "$REPO_ROOT/internal/classic/assets/scummtr-linux-x64"
cp "$LINUX_SRC/scummrp" "$REPO_ROOT/internal/charset/assets/scummrp-linux-x64"
echo "  [linux] internal assets updated"

# --- macOS ---
echo ""
echo "  [darwin] Downloading..."
curl -sL "${BASE_URL}/scummtr-${SCUMMTR_VERSION}-macos.zip" \
    -o "$TMPDIR_DL/scummtr-macos.zip"
unzip -q "$TMPDIR_DL/scummtr-macos.zip" -d "$TMPDIR_DL/macos"

mkdir -p "$REPO_ROOT/bin/darwin"
for tool in scummtr scummrp scummfont FontXY; do
    bin="$(find "$TMPDIR_DL/macos" -name "$tool" -not -name "*.zip" | head -1)"
    cp "$bin" "$REPO_ROOT/bin/darwin/$tool"
    chmod +x "$REPO_ROOT/bin/darwin/$tool"
done
echo "  [darwin] bin/darwin/ updated"

DARWIN_SCUMMTR="$(find "$TMPDIR_DL/macos" -name "scummtr" | head -1)"
DARWIN_SCUMMRP="$(find "$TMPDIR_DL/macos" -name "scummrp" | head -1)"
cp "$DARWIN_SCUMMTR" "$REPO_ROOT/internal/classic/assets/scummtr-darwin-x64"
cp "$DARWIN_SCUMMRP" "$REPO_ROOT/internal/charset/assets/scummrp-darwin-x64"
echo "  [darwin] internal assets updated"

# --- Windows ---
echo ""
echo "  [windows] Downloading..."
curl -sL "${BASE_URL}/scummtr-${SCUMMTR_VERSION}-win32.zip" \
    -o "$TMPDIR_DL/scummtr-win32.zip"
unzip -q "$TMPDIR_DL/scummtr-win32.zip" -d "$TMPDIR_DL/win32"

cp "$TMPDIR_DL/win32/scummtr-${SCUMMTR_VERSION}-win32/scummtr.exe" \
    "$REPO_ROOT/internal/classic/assets/scummtr-windows-x64.exe"
cp "$TMPDIR_DL/win32/scummtr-${SCUMMTR_VERSION}-win32/scummrp.exe" \
    "$REPO_ROOT/internal/charset/assets/scummrp-windows-x64.exe"
echo "  [windows] internal assets updated"

# --- Summary ---
echo ""
echo "=== Done. Files updated: ==="
echo ""
echo "  bin/linux/:                    $(ls "$REPO_ROOT/bin/linux/" | tr '\n' ' ')"
echo "  bin/darwin/:                   $(ls "$REPO_ROOT/bin/darwin/" | tr '\n' ' ')"
echo "  internal/classic/assets/:      $(ls "$REPO_ROOT/internal/classic/assets/" | tr '\n' ' ')"
echo "  internal/charset/assets/:      $(ls "$REPO_ROOT/internal/charset/assets/" | tr '\n' ' ')"
echo ""
echo "Commit bin/ and internal/*/assets/ to keep them in sync."
