#!/usr/bin/env bash
# init_translation.sh — Initialize a swedish.txt translation file for the active game.
#
# The active game is determined by pwd (must be inside games/<game>/).
#
# Reads the extracted english.txt and writes a swedish.txt where every
# non-blank line is prefixed with [E] so it can be identified as untranslated
# during review. Lines are blank where the game stores empty-content strings;
# these must stay blank to preserve positional alignment with the game data.
#
# The script refuses to run if swedish.txt already exists and contains any
# lines that are not blank and not [E]-prefixed — i.e. if real translation work
# is already present.
#
# Usage:
#   cd games/monkey1 && bash ../../scripts/init_translation.sh

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
detect_game "${1:-}"

ENGLISH="$GAME_GEN/strings/english.txt"
SWEDISH="$GAME_TRANSLATION/swedish.txt"

if [[ ! -f "$ENGLISH" ]]; then
    echo "ERROR: $ENGLISH not found." >&2
    echo "Run 'bash scripts/extract.sh' first to extract game assets." >&2
    exit 1
fi

# Safety check: refuse to overwrite existing translated content.
if [[ -f "$SWEDISH" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
        # A line with real translation content: non-blank and not [E]-prefixed
        if [[ -n "$line" && "$line" != \[E\]* ]]; then
            echo "ERROR: $SWEDISH already contains translated content." >&2
            echo "       Delete it manually if you really want to start over." >&2
            exit 1
        fi
    done < "$SWEDISH"
    echo "NOTE: $SWEDISH exists but contains only [E]-prefixed or blank lines — reinitialising."
fi

mkdir -p "$(dirname "$SWEDISH")"

# Write swedish.txt: prefix non-blank lines with [E], leave blank lines blank.
added=0
total=0
while IFS= read -r line || [[ -n "$line" ]]; do
    total=$((total + 1))
    if [[ -n "$line" ]]; then
        printf '[E]%s\n' "$line"
        added=$((added + 1))
    else
        printf '\n'
    fi
done < "$ENGLISH" > "$SWEDISH"

echo "Wrote $SWEDISH"
echo "  $added strings marked [E] (untranslated), $((total - added)) blank slots preserved."
echo ""
echo "Next step: open $GAME_TRANSLATION/swedish.txt and replace [E]-prefixed lines"
echo "with Swedish translations, removing the [E] prefix as you go."
