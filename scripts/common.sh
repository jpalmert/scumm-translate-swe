#!/usr/bin/env bash
# common.sh — Shared helpers for all scripts.
#
# Source this at the top of every script:
#   source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
#
# Provides:
#   REPO_ROOT   — absolute path to the repository root
#   detect_game — resolve the active game name from pwd or argument

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# detect_game [game_name]
#
# Resolves the active game. If an explicit game name is given, uses that.
# Otherwise, looks for a games/<game>/ ancestor in the current working
# directory. Exits with error if neither is found.
#
# Sets: GAME, GAME_DIR, GAME_GAME, GAME_GEN, GAME_DIST, GAME_TRANSLATION
detect_game() {
    local explicit="${1:-}"

    if [[ -n "$explicit" ]]; then
        GAME="$explicit"
    else
        # Try to infer from pwd: look for games/<name>/ in the path
        local cwd
        cwd="$(pwd)"
        if [[ "$cwd" == "$REPO_ROOT/games/"* ]]; then
            # Strip repo root + "games/" prefix, then take the first path component
            local rel="${cwd#$REPO_ROOT/games/}"
            GAME="${rel%%/*}"
        else
            echo "ERROR: Cannot determine which game to operate on." >&2
            echo "  Either cd into a games/<game>/ directory, or pass the game name as an argument." >&2
            exit 1
        fi
    fi

    if [[ -z "$GAME" ]]; then
        echo "ERROR: Game name is empty." >&2
        exit 1
    fi

    GAME_DIR="$REPO_ROOT/games/$GAME"
    GAME_GAME="$GAME_DIR/game"
    GAME_GEN="$GAME_DIR/gen"
    GAME_DIST="$GAME_DIR/dist"
    GAME_TRANSLATION="$GAME_DIR/translation"
}
