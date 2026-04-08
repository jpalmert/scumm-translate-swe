# Graphics Text Audit Report

**Status**: ✅ **COMPLETE**  
**Date**: 2026-04-08

---

## Findings

- **83 room backgrounds** decoded from CD-ROM version
- **4 rooms contain hardcoded text** (all proper nouns)
- **Decision**: Keep English names as-is

### Graphics with Text

1. **LFLF_0010** - "LUCASFILM GAMES" logo
2. **LFLF_0012** - "Mêlée Island" location name
3. **LFLF_0033** - "SCUMM BAR" bar name
4. **LFLF_0059** - "STAN'S PREVIOUSLY OWNED VESSELS" business name

All are proper nouns that remain in English (confirmed by monkeycd_swe).

---

## Tools Created

### `tools/decode_room.py`
Decodes SCUMM v5 room backgrounds (RMIM → SMAP) to PNG.
- All v5 codecs: RAW256 (1), ZIGZAG_V/H (14-48), MAJMIN (64-128)
- Usage: `python3 tools/decode_room.py <LFLF_dir> <output.png>`

### `tools/decode_object.py`
Decodes SCUMM v5 object images (OBIM → IM01 → SMAP) to PNG.
- Optional palette support via room CLUT
- Usage: `python3 tools/decode_object.py <OBIM_file> <output.png> [room_dir]`

---

## Assets

- **All rooms**: `/tmp/all_rooms/` (83 rooms)
- **Graphics with text**: `/tmp/rooms_with_text/` (4 confirmed + docs)

---

## Conclusion

**No graphics translation required** for Swedish fan translation.

All text in graphics is proper nouns (names, locations, businesses) that stay in English.

**Translation focus**:
- String resources via scummtr
- Character set fonts (Swedish diacriticals)

See `docs/GRAPHICS_WITH_TEXT.md` for translator documentation.
