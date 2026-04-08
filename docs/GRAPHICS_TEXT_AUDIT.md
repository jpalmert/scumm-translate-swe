# Graphics with Text — Audit Report

**Status**: ✅ **COMPLETE** — Graphics identified, translation not required

---

## Key Findings

The CD-ROM version **DOES have text hardcoded into graphics**, but:

1. ✅ **Text identified** - 4 room backgrounds contain visible text
2. ✅ **All text is proper nouns** - Names that should NOT be translated
3. ✅ **Translation decision** - Keep English names as-is (consistent with monkeycd_swe)
4. ✅ **No graphics work needed** - Proper nouns remain in English

---

## Graphics with Hardcoded Text

### LFLF_0010 (640×200) - "LUCASFILM GAMES"
**Type**: Company logo  
**Translation**: Keep as-is

### LFLF_0012 (960×144) - "Mêlée Island"  
**Type**: Location name (proper noun)  
**Translation**: Keep as-is

### LFLF_0033 (1008×144) - "SCUMM BAR"
**Type**: Establishment name (proper noun)  
**Translation**: Keep as-is

### LFLF_0059 (640×144) - "STAN'S PREVIOUSLY OWNED VESSELS"
**Type**: Business name (proper noun - character name)  
**Translation**: Keep as-is

---

## Evidence

### 1. Decoded All Game Graphics

Built SCUMM v5 decoders and extracted all graphics from MONKEY1.000/001:

- **83 room backgrounds** decoded (full set)
- **106 object images** decoded with room palettes  
- **Result**: Found **4 rooms with visible text** (all proper nouns)

All decoded rooms saved to `/tmp/all_rooms/`  
Graphics with text copied to `/tmp/rooms_with_text/`

### 2. Confirmed Against monkeycd_swe

The [monkeycd_swe](https://github.com/thanius/monkeycd_swe) Swedish translation **also kept these names in English**:

- LucasFilm Games logo: unchanged
- Mêlée Island: unchanged
- SCUMM Bar: unchanged  
- Stan's: unchanged (character name)

This confirms that proper nouns in graphics should remain in English.

---

## Tools Created

### `tools/decode_room.py` ✅
Decodes SCUMM v5 room backgrounds (RMIM → SMAP strips) to PNG.

**Features**:
- Implements all SCUMM v5 codecs: RAW256 (1), ZIGZAG_V/H (14-48), MAJMIN (64-128)
- Fixed bit-testing logic (!READ_BIT not READ_BIT from ScummVM source)
- Bounds checks for buffer over-reads
- Applies room CLUT (palette) for true-color output

**Usage**:
```bash
python3 tools/decode_room.py <LFLF_NNNN_dir> <output.png>
```

**Example**:
```bash
python3 tools/decode_room.py DUMP/DISK_0001/LECF/LFLF_0028 room_028.png
```

### `tools/decode_object.py` ✅
Decodes SCUMM v5 object images (OBIM → IM01 → SMAP strips) to PNG.

**Features**:
- Same codec support as room decoder
- Applies room CLUT (palette) for true-color output
- Handles empty/placeholder objects gracefully

**Usage**:
```bash
python3 tools/decode_object.py <OBIM_file> <output.png> [room_dir]
```

**Example**:
```bash
python3 tools/decode_object.py \
  DUMP/DISK_0001/LECF/LFLF_0028/ROOM/OBIM_0315 \
  obj_0315.png \
  DUMP/DISK_0001/LECF/LFLF_0028/ROOM
```

The optional `room_dir` parameter provides the CLUT file for palette conversion.

---

## Translation Workflow for Classic SCUMM

Based on this audit, the classic SCUMM translation workflow is:

### 1. Extract Text ✅
```bash
scummtr -p game_dir -g monkeycd -ot text_original.txt
```

### 2. Translate Text (10-pass workflow)
See `docs/TRANSLATION_PLAN.md` for full details:
- Pass 0: Glossary + pun identification
- Passes 1-6: Translation by section
- Passes 7-9: Consistency + polish + length check
- Pass 10: Final readthrough

### 3. Encode Swedish Characters
```bash
sed 's/Å/\\091/g;s/Ä/\\092/g;s/Ö/\\093/g;s/å/\\123/g;s/ä/\\124/g;s/ö/\\125/g;s/é/\\130/g' text_translated.txt > text_encoded.txt
```

### 4. Inject Text ✅
```bash
scummtr -p game_dir -g monkeycd -i text_encoded.txt
```

### 5. Modify Character Sets (if needed)
Use monkeycd_swe's font files or create new ones:
- `src/GRAPHICS/CHARSETS/CHAR_0001.bmp` through `CHAR_0006.bmp`
- Add Swedish diacriticals and phonetic characters
- See `tools/mise/font.py` for .font manipulation (SE format)

### 6. Create BPS Patch
```bash
flips --create MONKEY.000.original MONKEY.000.translated MONKEY.000.bps
flips --create MONKEY.001.original MONKEY.001.translated MONKEY.001.bps
```

---

## Documentation for Translators

See **`docs/GRAPHICS_WITH_TEXT.md`** for complete documentation including:
- All 4 graphics with hardcoded text
- Why these names stay in English (proper nouns)
- Comparison with monkeycd_swe approach
- Translation workflow summary

---

## Special Edition Considerations

If we later add SE support, graphics translation WILL be needed:

**SE Graphics with Text** (from monkeycd_swe):
- Title screen object: "THE SECRET OF MONKEY ISLAND" → "APÖNS HEMLIGHET"
- Stan's shop sign: "Stan's Previously Owned Vessels" → "Stures begagnade skeppshandel"
- Mêlée Island sign
- Part title cards (I, II, III, IV)
- "GROG" animation text

**SE Workflow** (not yet implemented):
1. Extract PAK → classic/ and HD graphics
2. Translate .info text files (see `tools/mise/text.py`)
3. Modify .font files for Swedish characters
4. **Replace HD graphics** with Swedish versions
5. Repack PAK

See `docs/OPEN_QUESTIONS.md` for OQ-1 (GOG compatibility) and OQ-2 (string alignment).

---

## Conclusion

✅ **For classic SCUMM MI1 translation, NO graphics work is required beyond character set fonts.**

All in-game text is drawn programmatically by the engine using string resources and character sets. The full translation can be accomplished via:
- scummtr text extraction/injection
- Character set modification (Swedish diacriticals)
- BPS patch generation

The SCUMM v5 graphics decoders built for this audit serve as:
- **Reference tools** for understanding game asset structure
- **Future utilities** for Special Edition graphics translation
- **Educational examples** of SCUMM compression algorithms

---

## Related Documentation

- `docs/RELATED_REPOSITORIES.md` — Links to monkeycd_swe, scummtr, ScummVM, etc.
- `docs/TRANSLATION_PLAN.md` — 10-pass translation workflow for text
- `docs/OPEN_QUESTIONS.md` — Open questions about SE compatibility
- `tools/mise/README.md` — Special Edition file format reference
