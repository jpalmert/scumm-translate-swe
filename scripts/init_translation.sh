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
SKIP_SWEDISH=false
if [[ -f "$SWEDISH" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
        # A line with real translation content: non-blank and not [E]-prefixed
        if [[ -n "$line" && "$line" != \[E\]* ]]; then
            echo "NOTE: $SWEDISH already contains translated content — skipping."
            SKIP_SWEDISH=true
            break
        fi
    done < "$SWEDISH"
    if ! $SKIP_SWEDISH; then
        echo "NOTE: $SWEDISH exists but contains only [E]-prefixed or blank lines — reinitialising."
    fi
fi

mkdir -p "$(dirname "$SWEDISH")"

if ! $SKIP_SWEDISH; then
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
fi

# --- Initialise SE UI text translation ---
UITEXT_ENGLISH="$GAME_GEN/strings/uitext_english.txt"
UITEXT_SWEDISH="$GAME_TRANSLATION/uitext_swedish.txt"

if [[ -f "$UITEXT_ENGLISH" ]]; then
    # Safety check: refuse to overwrite existing translated content.
    if [[ -f "$UITEXT_SWEDISH" ]]; then
        has_translated=false
        while IFS= read -r line || [[ -n "$line" ]]; do
            if [[ -n "$line" && "$line" != \#* ]]; then
                # Check if the value part (after first tab) starts with [E]
                val="${line#*	}"
                if [[ "$val" != "[E]"* && "$val" != "$line" ]]; then
                    has_translated=true
                    break
                fi
            fi
        done < "$UITEXT_SWEDISH"
        if $has_translated; then
            echo ""
            echo "NOTE: $UITEXT_SWEDISH already contains translated content — skipping."
        else
            echo ""
            echo "NOTE: $UITEXT_SWEDISH exists but has no translations — reinitialising."
        fi
    fi

    if [[ ! -f "$UITEXT_SWEDISH" ]] || ! $has_translated 2>/dev/null; then
        uitext_added=0
        {
            echo "# SE UI Text Swedish Translation"
            echo "# Format: KEY<TAB>Swedish text (or [E]English text if untranslated)"
            echo "#"
            while IFS= read -r line || [[ -n "$line" ]]; do
                [[ "$line" == \#* ]] && continue
                [[ -z "$line" ]] && continue
                key="${line%%	*}"
                val="${line#*	}"
                printf '%s\t[E]%s\n' "$key" "$val"
                uitext_added=$((uitext_added + 1))
            done < "$UITEXT_ENGLISH"
        } > "$UITEXT_SWEDISH"
        echo ""
        echo "Wrote $UITEXT_SWEDISH"
        echo "  $uitext_added strings marked [E] (untranslated)."
    fi
fi

# --- Initialise SE hints translation ---
HINTS_ENGLISH="$GAME_GEN/strings/hints_english.txt"
HINTS_SWEDISH="$GAME_TRANSLATION/hints_swedish.txt"

if [[ -f "$HINTS_ENGLISH" ]]; then
    # Safety check: refuse to overwrite existing translated content.
    if [[ -f "$HINTS_SWEDISH" ]]; then
        has_translated=false
        while IFS= read -r line || [[ -n "$line" ]]; do
            if [[ -n "$line" && "$line" != \#* ]]; then
                # Hints format: ADDR<TAB>Swedish text
                # Check if the last field starts with [E]
                val="${line#*	}"
                if [[ -n "$val" && "$val" != "[E]"* ]]; then
                    has_translated=true
                    break
                fi
            fi
        done < "$HINTS_SWEDISH"
        if $has_translated; then
            echo ""
            echo "NOTE: $HINTS_SWEDISH already contains translated content — skipping."
        else
            echo ""
            echo "NOTE: $HINTS_SWEDISH exists but has no translations — reinitialising."
        fi
    fi

    if [[ ! -f "$HINTS_SWEDISH" ]] || ! $has_translated 2>/dev/null; then
        hints_added=0
        {
            echo "# SE Hint Text Swedish Translation"
            echo "# Format: ADDR<TAB>Swedish text (or [E]English text if untranslated)"
            echo "#"
            while IFS= read -r line || [[ -n "$line" ]]; do
                [[ "$line" == \#* ]] && continue
                [[ -z "$line" ]] && continue
                # Format: ADDR<TAB>text
                addr="${line%%	*}"
                text="${line#*	}"
                printf '%s\t[E]%s\n' "$addr" "$text"
                hints_added=$((hints_added + 1))
            done < "$HINTS_ENGLISH"
        } > "$HINTS_SWEDISH"
        echo ""
        echo "Wrote $HINTS_SWEDISH"
        echo "  $hints_added strings marked [E] (untranslated)."
    fi
fi

echo ""
echo "Next step: open $GAME_TRANSLATION/swedish.txt and replace [E]-prefixed lines"
echo "with Swedish translations, removing the [E] prefix as you go."
