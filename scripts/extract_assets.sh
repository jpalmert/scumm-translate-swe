#!/usr/bin/env bash
# extract_assets.sh — Extract English charset assets and dialog strings from MI1 game files.
#
# Works from a directory containing the classic SCUMM data files.
# Accepts both naming conventions:
#   MONKEY1.000 / MONKEY1.001  — SE embedded classic (game ID: monkeycdalt)
#   MONKEY.000  / MONKEY.001   — Classic CD version  (game ID: monkeycd)
# Both upper and lowercase filenames are accepted.
#
# Reads:   game dir containing MONKEY1.000/001 or MONKEY.000/001
# Writes:  game/monkey1/gen/charset/english/CHAR_NNNN  — raw CHAR font blocks
#          game/monkey1/gen/charset/english_bitmaps/*.bmp — visual reference for editing Swedish glyphs
#          game/monkey1/gen/strings/english.txt          — dialog strings for translation
#
# All outputs live under game/ which is gitignored.
#
# Usage (from repo root):
#   bash scripts/extract_assets.sh [game_dir]
#   Default game_dir: game/monkey1/
#
# Prerequisites:
#   bash scripts/install_deps.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SCUMMRP="$REPO_ROOT/bin/linux/scummrp"
SCUMMFONT="$REPO_ROOT/bin/linux/scummfont"
SCUMMTR="$REPO_ROOT/bin/linux/scummtr"
if [[ "$(uname)" == "Darwin" ]]; then
    SCUMMRP="$REPO_ROOT/bin/darwin/scummrp"
    SCUMMFONT="$REPO_ROOT/bin/darwin/scummfont"
    SCUMMTR="$REPO_ROOT/bin/darwin/scummtr"
fi

for bin in "$SCUMMRP" "$SCUMMFONT" "$SCUMMTR"; do
    if [[ ! -x "$bin" ]]; then
        echo "ERROR: $(basename "$bin") not found. Run: bash scripts/install_deps.sh" >&2
        exit 1
    fi
done

INPUT_DIR="${1:-$REPO_ROOT/game/monkey1}"

if [[ ! -d "$INPUT_DIR" ]]; then
    echo "ERROR: directory not found: $INPUT_DIR" >&2
    exit 1
fi

# --- Detect game variant and normalise filenames into a temp work dir ---
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
GAME_DIR="$WORK/game"
mkdir -p "$GAME_DIR"

GAME_ID=""

# SE embedded classic: MONKEY1.000 / MONKEY1.001
for name in MONKEY1 monkey1; do
    if [[ -f "$INPUT_DIR/$name.000" && -f "$INPUT_DIR/$name.001" ]]; then
        cp "$INPUT_DIR/$name.000" "$GAME_DIR/MONKEY1.000"
        cp "$INPUT_DIR/$name.001" "$GAME_DIR/MONKEY1.001"
        GAME_ID="monkeycdalt"
        break
    fi
done

# Classic CD: MONKEY.000 / MONKEY.001
if [[ -z "$GAME_ID" ]]; then
    for name in MONKEY monkey; do
        if [[ -f "$INPUT_DIR/$name.000" && -f "$INPUT_DIR/$name.001" ]]; then
            cp "$INPUT_DIR/$name.000" "$GAME_DIR/MONKEY.000"
            cp "$INPUT_DIR/$name.001" "$GAME_DIR/MONKEY.001"
            GAME_ID="monkeycd"
            break
        fi
    done
fi

if [[ -z "$GAME_ID" ]]; then
    echo "ERROR: No MONKEY1.000/001 or MONKEY.000/001 found in $INPUT_DIR" >&2
    exit 1
fi

echo "Detected game variant: $GAME_ID"

GEN_ROOT="$REPO_ROOT/game/monkey1/gen"

# --- Full block dump (used for CHAR, room images, and object images) ---
echo ""
echo "=== Dumping all game blocks ==="
DUMP_DIR="$WORK/full_dump"
"$SCUMMRP" -g "$GAME_ID" -p "$GAME_DIR" -od "$DUMP_DIR"

# Locate the directory containing the CHAR blocks — differs between game variants
CHAR_DIR="$(find "$DUMP_DIR" -name "CHAR_0001" -exec dirname {} \; 2>/dev/null | head -1)"
if [[ -z "$CHAR_DIR" ]]; then
    echo "ERROR: CHAR blocks not found in scummrp output" >&2
    exit 1
fi

# --- Save raw CHAR blocks (used as templates by build.sh) ---
echo ""
echo "=== Saving CHAR blocks ==="
CHAR_OUT="$GEN_ROOT/charset/english"
mkdir -p "$CHAR_OUT"
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    src="$CHAR_DIR/$n"
    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not in dump"
        continue
    fi
    cp "$src" "$CHAR_OUT/$n"
    echo "  $n -> $CHAR_OUT/$n"
done

# --- Export BMPs (visual reference for editing Swedish glyph bitmaps) ---
echo ""
echo "=== Exporting BMPs ==="
BMP_OUT="$GEN_ROOT/charset/english_bitmaps"
mkdir -p "$BMP_OUT"
for n in CHAR_0001 CHAR_0002 CHAR_0003 CHAR_0004 CHAR_0006; do
    src="$CHAR_DIR/$n"
    if [[ ! -f "$src" ]]; then
        echo "  SKIP $n: not in dump"
        continue
    fi
    "$SCUMMFONT" o "$src" "$BMP_OUT/$n.bmp"
    echo "  $n -> $BMP_OUT/$n.bmp"
done

# --- Extract dialog strings ---
echo ""
echo "=== Extracting strings ==="
STR_OUT="$GEN_ROOT/strings"
mkdir -p "$STR_OUT"
"$SCUMMTR" -g "$GAME_ID" -p "$GAME_DIR" -hI -A aov -o -f "$STR_OUT/english.txt"

# Post-process extracted strings into clean UTF-8 for translators:
#   1. Replace ^ (SCUMM ellipsis byte 0x5E) with ...
#   2. Strip trailing @ padding (scummtr pads fixed-width text slots with @).
#   3. Convert SCUMM character escape codes to their UTF-8 characters:
#        \130 = é,  \136 = ê,  \015 = ®,  \250 = non-breaking space
sed -i \
    -e '/^;;/d' \
    -e 's/\^/.../g' \
    -e 's/@\+$//' \
    -e 's/\\130/é/g' \
    -e 's/\\136/ê/g' \
    -e 's/\\015/®/g' \
    -e 's/\\250/\xc2\xa0/g' \
    "$STR_OUT/english.txt"

echo "  -> $STR_OUT/english.txt ($(wc -l < "$STR_OUT/english.txt") lines)"


# --- Decode room backgrounds (requires Pillow: pip install Pillow) ---
echo ""
echo "=== Decoding room backgrounds ==="
ROOMS_OUT="$GEN_ROOT/rooms"
mkdir -p "$ROOMS_OUT"
LECF_DIR="$(find "$DUMP_DIR" -type d -name "LECF" | head -1)"
if [[ -z "$LECF_DIR" ]]; then
    echo "  SKIP: LECF directory not found in dump"
else
    count=0
    for lflf_dir in "$LECF_DIR"/LFLF_*; do
        [[ -d "$lflf_dir" ]] || continue
        room_num="${lflf_dir##*_}"
        out_png="$ROOMS_OUT/room_$room_num.png"
        [[ -f "$out_png" ]] && continue  # skip if already decoded
        if python3 "$REPO_ROOT/tools/decode_room.py" "$lflf_dir" "$out_png" 2>/dev/null; then
            count=$((count + 1))
        fi
    done
    echo "  $count room backgrounds -> $ROOMS_OUT"
fi

# --- Decode object images (requires Pillow: pip install Pillow) ---
echo ""
echo "=== Decoding object images ==="
OBJECTS_OUT="$GEN_ROOT/objects"
mkdir -p "$OBJECTS_OUT"
if [[ -z "$LECF_DIR" ]]; then
    echo "  SKIP: LECF directory not found in dump"
else
    count=0
    for lflf_dir in "$LECF_DIR"/LFLF_*; do
        [[ -d "$lflf_dir" ]] || continue
        room_num="${lflf_dir##*_}"
        room_obj_dir="$OBJECTS_OUT/room_$room_num"
        # Find all OBIM files in this room's OI directories
        while IFS= read -r -d '' obim_file; do
            obj_dir="$(dirname "$obim_file")"
            obj_num="$(basename "$obj_dir")"
            out_png="$room_obj_dir/${obj_num}.png"
            mkdir -p "$room_obj_dir"
            [[ -f "$out_png" ]] && continue  # skip if already decoded
            if python3 "$REPO_ROOT/tools/decode_object.py" "$obim_file" "$out_png" 2>/dev/null; then
                count=$((count + 1))
            fi
        done < <(find "$lflf_dir" -name "OBIM" -print0 2>/dev/null)
    done
    echo "  $count object images -> $OBJECTS_OUT"
fi

echo ""
echo "Done."
