# tools/ — Developer utilities

Standalone Python 3 tools for inspecting MI1SE files. Not part of the main build pipeline.

## pak.py — PAK archive extractor/repacker

Inspect and unpack the MI1SE PAK archive, or repack a modified directory back into a PAK.

```bash
python3 tools/pak.py extract Monkey1.pak output_dir/ [game]
python3 tools/pak.py repack  output_dir/ output.pak original.pak [game]
```

`game`: 1 for MI1SE, 2 for MI2SE (auto-detected from filename if omitted).

## patch_verbs.py — Verb button layout patcher

Patches verb button X/Y coordinates in SCRP_0022. Can be run standalone for inspection.

```bash
python3 tools/patch_verbs.py <input_scrp_0022> <output_scrp_0022>
```
