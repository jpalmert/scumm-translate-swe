#!/usr/bin/env bash
# init_translation.sh — Initialize a swedish.txt translation file for a game.
#
# Usage: bash scripts/init_translation.sh <game>
#   game  — subdirectory name under game/ and translation/ (e.g. monkey1)
#
# Reads the extracted english.txt and writes a swedish.txt where every
# non-blank line is prefixed with [E] so it can be identified as untranslated
# during review. Lines are blank where the game stores empty-content strings;
# these must stay blank to preserve positional alignment with the game data.
#
# The script refuses to run if swedish.txt already exists and contains any
# lines that are not blank and not [E]-prefixed — i.e. if real translation work
# is already present.
set -euo pipefail

GAME="${1:-}"
if [[ -z "$GAME" ]]; then
    echo "Usage: $0 <game>"
    echo "  e.g. $0 monkey1"
    exit 1
fi

ENGLISH="game/$GAME/gen/strings/english.txt"
SWEDISH="translation/$GAME/swedish.txt"

if [[ ! -f "$ENGLISH" ]]; then
    echo "ERROR: $ENGLISH not found."
    echo "Run 'bash scripts/extract.sh' first to extract game assets."
    exit 1
fi

# Safety check: refuse to overwrite existing translated content.
if [[ -f "$SWEDISH" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
        # A line with real translation content: non-blank and not [E]-prefixed
        if [[ -n "$line" && "$line" != \[E\]* ]]; then
            echo "ERROR: $SWEDISH already contains translated content."
            echo "       Delete it manually if you really want to start over."
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
echo "Next step: open translation/$GAME/swedish.txt and replace [E]-prefixed lines"
echo "with Swedish translations, removing the [E] prefix as you go."
