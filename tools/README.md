# tools/ — Developer utilities

These are standalone Python 3 tools for inspecting and working with MI1SE files.
They are not part of the main build pipeline — the Go patcher handles all end-user patching.

## pak.py — PAK archive extractor/repacker

Inspect and unpack the MI1SE PAK archive, or repack a modified directory back into a PAK.

```bash
python3 tools/pak.py extract Monkey1.pak output_dir/ [game]
python3 tools/pak.py repack  output_dir/ output.pak original.pak [game]
```

`game`: 1 for MI1SE, 2 for MI2SE (auto-detected from filename if omitted).

## patch_verbs.py — Verb button patcher

Patches Swedish verb button labels in the SE UI. Used internally by the Go patcher
during the SE patching pipeline; can also be run standalone for inspection.

## Requirements

```bash
pip install Pillow
```
