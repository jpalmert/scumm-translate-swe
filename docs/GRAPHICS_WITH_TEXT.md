# Graphics with Hardcoded Text

**Status**: Identified and documented  
**Decision**: Keep English names as-is (proper nouns)  
**Date**: 2026-04-08

---

## Overview

The Monkey Island 1 CD-ROM version has **text hardcoded into graphics** in several room backgrounds. These are pixel-art text embedded in PNG images, NOT programmatically drawn by the engine.

**Translation decision**: All names in graphics are **proper nouns** and will **remain in English**. These do not need translation.

---

## Graphics with Hardcoded Text

### 1. LFLF_0010 - LucasFilm Games Logo
**File**: `/tmp/rooms_with_text/LFLF_0010_LUCASFILM_GAMES.png`  
**Dimensions**: 640×200  
**Text**: "LUCASFILM GAMES" (metallic logo lettering)  
**Type**: Company logo  
**Action**: **Keep as-is** (company names never translated)

---

### 2. LFLF_0012 - Mêlée Island Overview
**File**: `/tmp/rooms_with_text/LFLF_0012_MELEE_ISLAND.png`  
**Dimensions**: 960×144  
**Text**: "Mêlée Island" (visible on sign/label)  
**Type**: Location name (proper noun)  
**Action**: **Keep as-is** (location names remain in English)

---

### 3. LFLF_0033 - SCUMM Bar Harbor
**File**: `/tmp/rooms_with_text/LFLF_0033_SCUMM_BAR_SIGN.png`  
**Dimensions**: 1008×144  
**Text**: "SCUMM BAR" (sign on harbor building)  
**Type**: Establishment name (proper noun)  
**Action**: **Keep as-is** (bar name is a proper noun)

---

### 4. LFLF_0059 - Stan's Ship Emporium
**File**: `/tmp/rooms_with_text/LFLF_0059_STANS_SHOP.png`  
**Dimensions**: 640×144  
**Text**: 
- "STAN'S" (large sign)
- "PREVIOUSLY OWNED VESSELS" (subtitle)

**Type**: Business name (proper noun)  
**Action**: **Keep as-is** (Stan is a character name, proper noun)

---

## Important Notes for Translators

### Why These Stay in English

1. **Proper nouns** - Character names (Stan), place names (Mêlée Island), and business names don't get translated
2. **Company logos** - LucasFilm Games is a trademark
3. **Consistency** - The game refers to these names in dialogue and UI text, which also use the English names
4. **Technical difficulty** - Modifying graphics is labor-intensive (requires image editing, font matching, pixel-art work)

### Graphics vs Programmatic Text

The game has two types of text:

| Type | Location | Translatable? | How to Translate |
|------|----------|---------------|------------------|
| **Programmatic text** | Dialogue, verbs, object names, descriptions | ✅ Yes | Via scummtr (text files) |
| **Graphics text** | Room backgrounds, signs, logos | ❌ No (proper nouns) | Not needed |

**All translatable text** is extracted via scummtr and appears in `text.swe` format. Graphics text is hardcoded pixel art and consists only of proper nouns.

---

## Comparison with monkeycd_swe

The [monkeycd_swe](https://github.com/thanius/monkeycd_swe) Swedish translation **also kept these names in English**:

| Our Room | monkeycd_swe | English Text | Swedish Translation |
|----------|--------------|--------------|---------------------|
| LFLF_0010 | ROOM_010 | LUCASFILM GAMES | (unchanged) |
| LFLF_0012 | ROOM_011 | Mêlée Island | (unchanged) |
| LFLF_0033 | ROOM_028 | SCUMM BAR | (unchanged) |
| LFLF_0059 | ROOM_049 | STAN'S | (unchanged - Stan is a character name) |

The 48 PNG files in monkeycd_swe are either:
1. Special Edition HD graphics (not present in classic SCUMM)
2. Custom visual enhancements
3. Character set modifications (fonts with Swedish diacriticals)

Classic SCUMM CD-ROM translation does not require graphics modification.

---

## Room Numbering Reference

Internal SCUMM room IDs differ from monkeycd_swe numbering:

| Our LFLF | monkeycd_swe | Description |
|----------|--------------|-------------|
| LFLF_0010 | ROOM_010 | LucasFilm logo |
| LFLF_0012 | ROOM_011 | Mêlée Island overview |
| LFLF_0033 | ROOM_028 | SCUMM Bar harbor |
| LFLF_0059 | ROOM_049 | Stan's shop |

---

## Translation Workflow Summary

### ✅ What to Translate (via scummtr)
- All dialogue
- Object names and descriptions
- Verb UI text
- Room descriptions
- System messages

### ❌ What NOT to Translate
- Graphics text (proper nouns only)
- Character names (Guybrush, LeChuck, Stan, etc.)
- Location names (Mêlée Island, Monkey Island)
- Business/ship names

### 🔧 What to Modify
- Character sets (fonts) - Add Swedish diacriticals (Å, Ä, Ö, å, ä, ö, é)

---

## Decoded Room Assets

All 83 room backgrounds have been decoded and saved:

- **Full set**: `/tmp/all_rooms/` (all 83 rooms)
- **Graphics with text**: `/tmp/rooms_with_text/` (4 confirmed + 8 candidates)
- **Tools used**: `tools/decode_room.py`, `tools/decode_object.py`

See `docs/RELATED_REPOSITORIES.md` for links to monkeycd_swe source and other references.

---

## Conclusion

**Graphics translation is NOT required** for a proper Swedish translation of Monkey Island 1 CD-ROM.

All text in graphics consists of proper nouns that remain in English. The monkeycd_swe project confirms this approach.

Focus translation efforts on:
1. String resources (scummtr extraction/injection)
2. Character set modifications (Swedish diacriticals)
3. Text length adjustments (Swedish text is ~15-20% longer)

See `docs/TRANSLATION_PLAN.md` for the full 10-pass translation workflow.
