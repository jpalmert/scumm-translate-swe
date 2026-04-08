# Graphics with Text — Audit Report

**Status**: **INCOMPLETE** — Need proper CD-ROM version game files

---

## Summary

The [monkeycd_swe](https://github.com/thanius/monkeycd_swe) Swedish translation includes **48 PNG graphics files** with translated Swedish text, including:

- Title screen: "APÖNS HEMLIGHET" (The Monkey's Secret)
- Stan's shop sign: "Stures begagnade skeppshandel"
- Part title cards (I, II, III, IV)
- Mêlée Island sign
- Various object labels and signs

This proves that **graphics translation IS required** for a complete Swedish localization.

However, the current extraction is from **incomplete/wrong game files** (MONKEY1.000/001, only 4.6MB — likely a demo or floppy version, not the full CD-ROM version that monkeycd_swe targets).

---

## Current Status: Tools Built

### `tools/decode_room.py` ✅
Decodes SCUMM v5 room backgrounds (RMIM → SMAP strips) to PNG.
- Implements all SCUMM v5 codecs: RAW256 (1), ZIGZAG_V/H (14-48), MAJMIN (64-128)
- Fixed bit-testing logic (!READ_BIT not READ_BIT)
- Adds bounds checks for buffer over-reads
- Usage: `python3 tools/decode_room.py <LFLF_NNNN_dir> <output.png>`

### `tools/decode_object.py` ✅
Decodes SCUMM v5 object images (OBIM → IM01 → SMAP strips) to PNG.
- Same codec support as room decoder
- Currently outputs **grayscale** (palette indices) — needs room CLUT for colors
- Usage: `python3 tools/decode_object.py <OBIM_file> <output.png>`

**Issue**: Object decoder outputs grayscale, making text hard to see. Need to:
1. Apply room-specific CLUT (palette) to get true colors
2. Or convert palette indices to colors based on CLUT block

---

## What We Know from monkeycd_swe

From the [monkeycd_swe SUMMARY.md](https://github.com/thanius/monkeycd_swe):

**Graphics with Swedish translations** (48 PNG files):
```
ROOM_009/ROOM_009_Object_0.png           — "APÖNS HEMLIGHET" (title graphic)
ROOM_010_Object_10.png                    — LucasFilm Games logo
ROOM_011.png, ROOM_027.png, ROOM_028.png  — Various room backgrounds
ROOM_041.png                              — ?
ROOM_046/COSTUME_006/                     — 18-frame animation
ROOM_049/ROOM_049_BKG.png                 — "Stures begagnade skeppshandel" sign
ROOM_049/ROOM_049_GROG/                   — "GROGG" animation (5 frames)
ROOM_049/ROOM_049_LANTERNS/               — Lantern frames (11 frames)
ROOM_054.png                              — Mêlée Island sign
ROOM_058.png, ROOM_061.png, ROOM_069.png  — ?
ROOM_082/ROOM_082_Object_44,69,75,84,89   — Multiple object sprites
PART_1234/ROOM_078-081.png                — Part title cards (I, II, III, IV)
```

**Key observations**:
- Many graphics are **object images** (OBIM), not room backgrounds (RMIM)
- Some are **costume animations** (COST) with multiple frames
- Part title cards are separate rooms (078-081)
- The README mentions "20 distinct game rooms" with modified graphics

---

## Blockers

### 1. Wrong Game Files
Currently extracted from `MONKEY1.000/001` (4.6MB) which is NOT the CD-ROM version.

**Need**: Full CD-ROM version files
- Size: ~20-30 MB combined (MONKEY.000 + MONKEY.001)
- Version: `monkeycd` or `monkeycdalt` (scummtr game ID)

**How to get**:
- Original CD-ROM (if available)
- GOG.com purchase: *The Secret of Monkey Island: Special Edition*  
  (contains classic SCUMM files in `classic/en/` subdirectory of Monkey1.pak)
- Steam purchase (same structure)

### 2. Grayscale Object Output
Object decoder doesn't apply room palette, so output is grayscale (palette indices).

**Options**:
1. **Modify decode_object.py** to read CLUT from room and apply colors
2. **Use scummvm-tools** if they have an image extractor (spoiler: they don't, scummrp only extracts raw blocks)
3. **Compare with monkeycd_swe PNGs** to identify which objects/rooms have text

### 3. Room Number Mismatch
Rooms 049 and 054 (mentioned in monkeycd_swe) don't exist in current extraction.

Likely cause: Different room numbering between game versions (floppy vs CD-ROM).

---

## Temporary Workaround: Use monkeycd_swe as Reference

Until we have proper game files, we can:

1. **Enumerate all graphics from monkeycd_swe** that have Swedish text
2. **Compare with English originals** in monkeycd_swe repo (if available) or extract from proper CD-ROM version
3. **Create translation map**: English text → Swedish text for each graphic
4. **Document graphics format** for each type (OBIM, COST, title cards)

The monkeycd_swe README notes that the graphics were translated but doesn't document HOW they were extracted/converted. Likely tools used:
- **ScummVM debugger** (built-in sprite/room viewer)
- **Custom extraction scripts** (not in repo)
- **Manual PNG editing** in GIMP/Photoshop after extraction

---

## Next Steps

### Immediate (Blocked by Game Files)
1. **Obtain proper CD-ROM version** of Monkey Island (GOG or Steam SE → extract classic/)
2. **Re-run full extraction** with scummrp on correct files
3. **Decode all objects** from rooms 009, 010, 028, 049, 054, 082
4. **Compare with monkeycd_swe PNGs** to identify English text

### Alternative (Can Do Now)
1. **Document all 48 graphics** from monkeycd_swe that were translated
2. **Extract Swedish text** from each PNG manually or via OCR
3. **Research original English text** by playing game in ScummVM or checking Let's Plays
4. **Create graphics translation list** for future encoding work

---

## Related Documentation

- `docs/RELATED_REPOSITORIES.md` — Links to monkeycd_swe, scummtr, ScummVM tools, etc.
- `docs/TRANSLATION_PLAN.md` — Multi-pass translation workflow for text
- `docs/OPEN_QUESTIONS.md` — OQ-1 (GOG vs Steam layout), OQ-2 (string ID alignment)
- `tools/mise/README.md` — Special Edition file format reference
