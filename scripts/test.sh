#!/usr/bin/env bash
# test.sh — Run all tests for the active game.
#
# The active game is determined by pwd (must be inside games/<game>/)
# or by passing the game name as an argument.
#
# Usage:
#   cd games/monkey1 && bash ../../scripts/test.sh          # unit tests only (no game files needed)
#   cd games/monkey1 && bash ../../scripts/test.sh --all    # include tests that need game files
#   bash scripts/test.sh monkey1                            # explicit game name
#   bash scripts/test.sh monkey1 --all                      # explicit + game-file tests
#
# Without --all: runs Go unit tests and Python tests (no game files or build
#   artifacts required).
# With --all: also runs buildpatcher asset tests and integration tests. These
#   require game files and build artifacts — missing files cause a FAIL.
#
# Exits with non-zero status if any test suite fails.

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Parse arguments: game name and --all flag
ALL=false
GAME_ARG=""
for arg in "$@"; do
    if [[ "$arg" == "--all" ]]; then
        ALL=true
    else
        GAME_ARG="$arg"
    fi
done

detect_game "$GAME_ARG"
echo "Game: $GAME"

# Find Go binary
GO_BIN=""
for candidate in go ~/go/bin/go /usr/local/go/bin/go; do
    if [[ "$candidate" == */* ]] && [[ -x "$candidate" ]] || [[ "$candidate" != */* ]] && command -v "$candidate" &>/dev/null; then
        GO_BIN="$candidate"
        break
    fi
done
if [[ -z "$GO_BIN" ]]; then
    echo "ERROR: Go not found. Install Go 1.21+ from https://go.dev/dl/" >&2
    exit 1
fi

cd "$REPO_ROOT"

failed=0

# --- Go unit tests ---
echo ""
echo "=== Go unit tests ==="
if "$GO_BIN" test ./... 2>&1; then
    echo "  PASS"
else
    echo "  FAIL"
    failed=1
fi

# --- Python tests ---
echo ""
echo "=== Python tests ==="
if python3 -m unittest discover -s tools -p 'test_*.py' 2>&1; then
    echo "  PASS"
else
    echo "  FAIL"
    failed=1
fi

# --- Tests that require game files (only with --all) ---
if $ALL; then
    # --- Go charset asset tests (buildpatcher) ---
    echo ""
    echo "=== Go charset asset tests (buildpatcher) ==="
    if ! ls internal/charset/gen/char_*_patched.bin &>/dev/null; then
        echo "  FAIL: no generated .bin files found — run build.sh first"
        failed=1
    elif "$GO_BIN" test -tags buildpatcher ./internal/charset/... 2>&1; then
        echo "  PASS"
    else
        echo "  FAIL"
        failed=1
    fi

    # --- Go integration tests ---
    echo ""
    echo "=== Go integration tests ==="
    if "$GO_BIN" test -tags integration ./... 2>&1; then
        echo "  PASS"
    else
        echo "  FAIL"
        failed=1
    fi
fi

# --- Summary ---
echo ""
if [[ $failed -eq 0 ]]; then
    if $ALL; then
        echo "All tests passed."
    else
        echo "All tests passed. (Run with --all to include game-file tests.)"
    fi
else
    echo "Some tests FAILED."
    exit 1
fi
