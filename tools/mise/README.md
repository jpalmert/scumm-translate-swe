# tools/mise — Monkey Island SE Translation Tools

Custom Python 3 replacement for MISETranslator. Handles the full SE translation
pipeline without requiring a GUI, Python 2.7, or PyQt4.

## Tools

### pak.py — PAK archive extractor/repacker
```bash
python3 pak.py extract Monkey1.pak output_dir/ [game]
python3 pak.py repack  output_dir/ output.pak original.pak [game]
```
`game`: 1 for MI:SE, 2 for MI2:SE (auto-detected from filename if omitted)

### text.py — .info text extractor/injector
```bash
python3 text.py extract speech.info speech.json
python3 text.py inject  speech.info speech.json speech_modified.info
```
Exports to JSON with `english` / `translation` fields. Fill in `translation`,
leave blank to keep the original English.

### font.py — .font glyph expander
```bash
python3 font.py inspect original.font
python3 font.py add-glyphs \
    --font original.font --png original_font.png \
    --glyphs new_chars.png \
    --map "Å:91,Ä:92,Ö:93,å:123,ä:124,ö:125,é:130" \
    --out-font modified.font --out-png modified_font.png
```
The `--glyphs` PNG should be a horizontal strip of the new characters,
one per glyph, on a transparent or solid background.

## Requirements
```bash
pip install Pillow
```

## SE Engine Limitation
Custom translations always replace the **French language slot**. After applying
a translated PAK, set the game language to French in the SE settings to see
the translation. This is an engine limitation, not a tool limitation.

## Supported file formats

| File | Game | Format |
|------|------|--------|
| speech.info | MI:SE | Fixed-stride, 3 languages, 256 bytes/slot |
| uiText.info | MI:SE | Fixed-stride, 3 languages, 256 bytes/slot |
| fr.speech.info | MI2:SE | Pointer-table, variable-length strings |
| fr.uitext.info | MI2:SE | Pointer-table, variable-length strings |
| *.hints.csv | both | Grouped hints (extraction pending) |
