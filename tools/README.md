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

## patch_verbs.py — Verb button layout patcher

Patches verb button X/Y coordinates in SCRP_0022. Can be run standalone for inspection.

```bash
python3 tools/patch_verbs.py <input_scrp_0022> <output_scrp_0022>
```
