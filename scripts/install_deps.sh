#!/usr/bin/env bash
# install_deps.sh — Re-download/rebuild developer tool dependencies
#
# NOTE: You do NOT normally need to run this script.
# The tool binaries (bin/) are committed to the repository and ready to use.
#
# Run this only if:
#   - The committed binaries don't work on your platform (e.g. macOS developer)
#   - You want to upgrade to a newer version of scummtr
#   - The bin/ directory is missing or corrupted
#
# Run from the repo root:
#   bash scripts/install_deps.sh
#
# What this installs/rebuilds:
#   bin/scummtr    — text extraction/injection for classic SCUMM games
#   bin/scummrp    — resource packer/unpacker (companion to scummtr)
#   bin/scummfont  — font tool (companion to scummtr)
#   bin/FontXY     — font positioning tool (companion to scummtr)
#
# scummtr is downloaded as a prebuilt binary for Linux and macOS (no cmake required).
# Python tools (tools/pak.py, tools/text.py) use only stdlib — no venv needed.
#
# Re-running this script is safe — it skips tools that are already present.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OS="$(uname -s)"

case "$OS" in
  Darwin) PLATFORM_DIR="$REPO_ROOT/bin/darwin" ;;
  *)      PLATFORM_DIR="$REPO_ROOT/bin/linux"  ;;
esac

mkdir -p "$PLATFORM_DIR"

# --- scummtr (prebuilt binary from GitHub releases) ---
#
# Prebuilt binaries for Linux (x64) and macOS from:
#   https://github.com/dwatteau/scummtr/releases
#
# This avoids needing cmake.

SCUMMTR_VERSION="0.5.1"
SCUMMTR_BASE_URL="https://github.com/dwatteau/scummtr/releases/download/v${SCUMMTR_VERSION}"

echo "=== scummtr ==="
if [ -f "$PLATFORM_DIR/scummtr" ]; then
    echo "  Already present: $PLATFORM_DIR/scummtr"
else
    echo "  Downloading scummtr v${SCUMMTR_VERSION}..."
    TMPDIR_SCUMMTR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR_SCUMMTR"' EXIT

    if [ "$OS" = "Darwin" ]; then
        curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-macos.zip" \
            -o "$TMPDIR_SCUMMTR/scummtr.zip"
        unzip -q "$TMPDIR_SCUMMTR/scummtr.zip" -d "$TMPDIR_SCUMMTR"
        for tool in scummtr scummrp scummfont FontXY; do
            bin="$(find "$TMPDIR_SCUMMTR" -name "$tool" -not -name "*.zip" | head -1)"
            if [ -n "$bin" ]; then
                cp "$bin" "$PLATFORM_DIR/$tool"
                chmod +x "$PLATFORM_DIR/$tool"
            fi
        done
    else
        curl -sL "${SCUMMTR_BASE_URL}/scummtr-${SCUMMTR_VERSION}-linux86.tar.gz" \
            | tar xz -C "$TMPDIR_SCUMMTR"
        for tool in scummtr scummrp scummfont FontXY; do
            cp "$TMPDIR_SCUMMTR/scummtr-${SCUMMTR_VERSION}-linux86/linux-x64/$tool" \
                "$PLATFORM_DIR/$tool"
            chmod +x "$PLATFORM_DIR/$tool"
        done
    fi
    echo "  Installed: $PLATFORM_DIR/scummtr (and scummrp, scummfont, FontXY)"
fi

# --- Summary ---

echo ""
echo "=== Summary ==="
echo "Tools in $PLATFORM_DIR/:"
ls "$PLATFORM_DIR/" | sed 's/^/  /'
echo ""
echo "Next step:"
echo "  Copy Monkey1.pak to game/monkey1/ then run:"
echo "  bash scripts/se/extract_classic_strings.sh"
echo ""
echo "All done."
