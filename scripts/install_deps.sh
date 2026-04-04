#!/usr/bin/env bash
# Install all dependencies for the SCUMM translation toolkit.
# Run once from the repo root: bash scripts/install_deps.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLS_DIR="$REPO_ROOT/tools"

echo "=== Installing system packages ==="
sudo apt-get update -q
sudo apt-get install -y \
    build-essential cmake git \
    wine wine64 \
    python3 python3-pip python3-venv \
    libgtk-3-dev pkg-config    # needed for flips GUI (optional)

echo ""
echo "=== Building scummtr (text extraction/injection tool) ==="
if [ ! -f "$TOOLS_DIR/bin/scummtr" ]; then
    cd "$TOOLS_DIR"
    git clone --depth=1 https://github.com/dwatteau/scummtr scummtr-src
    cd scummtr-src
    mkdir -p build && cd build
    cmake .. -DCMAKE_BUILD_TYPE=Release
    cmake --build . -- -j"$(nproc)"
    mkdir -p "$TOOLS_DIR/bin"
    cp scummtr scummrp scummfont FontXY "$TOOLS_DIR/bin/" 2>/dev/null || true
    echo "  scummtr built at $TOOLS_DIR/bin/scummtr"
    cd "$REPO_ROOT"
else
    echo "  scummtr already present, skipping"
fi

echo ""
echo "=== Building flips / Floating IPS (BPS patch creator) ==="
if [ ! -f "$TOOLS_DIR/bin/flips" ]; then
    cd "$TOOLS_DIR"
    git clone --depth=1 https://github.com/Alcaro/Flips flips-src
    cd flips-src
    # Build CLI-only version (no GTK required)
    g++ -O3 -DFLIPS_CLI \
        flips.cpp bps.cpp ips.cpp crc32.cpp \
        -o "$TOOLS_DIR/bin/flips"
    echo "  flips built at $TOOLS_DIR/bin/flips"
    cd "$REPO_ROOT"
else
    echo "  flips already present, skipping"
fi

echo ""
echo "=== Installing Python dependencies ==="
cd "$REPO_ROOT"
python3 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip -q
pip install pillow nutcracker -q
echo "  Pillow and nutcracker installed in .venv/"

echo ""
echo "=== Installing scummvm-tools (descumm, etc.) ==="
if ! command -v descumm &>/dev/null; then
    sudo apt-get install -y scummvm-tools 2>/dev/null || {
        echo "  scummvm-tools not in apt; building from source..."
        cd "$TOOLS_DIR"
        git clone --depth=1 https://github.com/scummvm/scummvm-tools scummvm-tools-src
        cd scummvm-tools-src
        cmake . -DCMAKE_BUILD_TYPE=Release
        make -j"$(nproc)"
        mkdir -p "$TOOLS_DIR/bin"
        cp descumm "$TOOLS_DIR/bin/" 2>/dev/null || true
        cd "$REPO_ROOT"
    }
else
    echo "  descumm already in PATH, skipping"
fi

echo ""
echo "=== Summary ==="
echo "Tools available in $TOOLS_DIR/bin/:"
ls "$TOOLS_DIR/bin/" 2>/dev/null || echo "  (none built yet)"
echo ""
echo "Python venv: source .venv/bin/activate"
echo ""
echo "NOTE: extractpak (for SE .pak extraction) must be obtained separately."
echo "      See tools/mise/README.md for SE workflow — pak.py replaces extractpak."
echo ""
echo "All done."
