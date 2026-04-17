# tools/ — Developer utilities

Standalone Python 3 tools for inspecting MI1SE files. `decode_room.py` and `decode_object.py`
are called automatically by `scripts/extract_assets.sh`; the others are standalone.

## pak.py — PAK archive extractor/repacker

Inspect and unpack the MI1SE PAK archive, or repack a modified directory back into a PAK.

```bash
python3 tools/pak.py extract Monkey1.pak output_dir/ [game]
python3 tools/pak.py repack  output_dir/ output.pak original.pak [game]
```

`game`: 1 for MI1SE, 2 for MI2SE (auto-detected from filename if omitted).

## decode_room.py — Room background decoder

Decodes a SCUMM v5 room background (RMIM block) to PNG. Called by `extract_assets.sh` for all
rooms; can also be run manually on a single LFLF directory from a scummrp dump.

```bash
python3 tools/decode_room.py game/monkey1/gen/full_dump/DISK_0001/LECF/LFLF_0028 room_028.png
```

Requires: `pip install Pillow`

## decode_object.py — Object image decoder

Decodes a SCUMM v5 object image (OBIM block) to PNG. Called by `extract_assets.sh` for all
objects; can also be run manually on a single OBIM file from a scummrp dump.

```bash
python3 tools/decode_object.py path/to/OBIM output.png
```

Requires: `pip install Pillow`

## calc_padding.py — Object name @ padding calculator

Calculates and optionally applies `@` padding to Swedish object names in `swedish.txt`.
Reads `dynamic_names.json` to find OBNA lines that get runtime replacements and pads
each name to fit the longest replacement. Called automatically by `scripts/build.sh`.

```bash
python3 tools/calc_padding.py                          # dry-run: show what would change
python3 tools/calc_padding.py --apply                  # apply padding in place
python3 tools/calc_padding.py --json PATH --translation PATH  # explicit paths
```

## find_dynamic_names.py — Runtime name-change extractor

Extracts `setObjectName` calls from MI1 SCUMM scripts by decompiling with `descumm`,
then builds a mapping of which scummtr lines replace which OBNA lines at runtime.
Called automatically by `scripts/extract_assets.sh`.

```bash
python3 tools/find_dynamic_names.py <game_dir> [output_json]
```

Requires: `scummrp` and `descumm` in `bin/<platform>/`

## patch_verbs.py — Verb button layout patcher

Patches verb button X/Y coordinates in SCRP_0022. Can be run standalone for inspection.

```bash
python3 tools/patch_verbs.py <input_scrp_0022> <output_scrp_0022>
```

## scumm_gfx.py — Shared SCUMM v5 graphics codec library

Shared module with block parsing helpers (`be32`, `le32`, `le16`, `find_block`) and
strip decoders (`decode_strip_basic`, `decode_strip_majmin`, `decode_strip_raw`,
`decode_strip`). Imported by `decode_room.py` and `decode_object.py`.
