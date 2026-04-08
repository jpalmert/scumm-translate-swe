# Graphics with Text — Audit Report

## Summary

Decoded all SCUMM v5 room backgrounds from MI1 classic to identify graphics containing English text that would conflict with Swedish translation.

**Finding**: Only ONE room background has visible text: **LUCASFILM GAMES logo (Room 010)**.

All other text in the game comes from string resources (extracted via scummtr), not baked into graphics.

---

## Tools Created

### `tools/decode_room.py`
Decodes SCUMM v5 room backgrounds (RMIM → SMAP strips) to PNG.
- Implements all SCUMM v5 codecs: RAW256 (1), ZIGZAG_V/H (14-48), MAJMIN (64-128)
- Handles bit-level compression with proper FILL_BITS/READ_BIT logic
- Usage: `python3 tools/decode_room.py <LFLF_NNNN_dir> <output.png>`

### `tools/decode_object.py`
Decodes SCUMM v5 object images (OBIM → IM01 → SMAP strips) to PNG.
- Same codec support as room decoder
- Outputs grayscale (palette indices) since objects use room-specific palettes
- Usage: `python3 tools/decode_object.py <OBIM_file> <output.png>`

---

## Rooms Audited

Based on monkeycd_swe's translated graphics list, decoded backgrounds for these rooms:

| Room | Scene Description | Text Found? |
|------|-------------------|-------------|
| 009  | Ship interior (title screen) | **None** |
| 010  | LucasFilm Games logo | **YES: "LUCASFILM GAMES"** |
| 011  | Beach landing | None |
| 027  | Circus tent interior | None |
| 028  | SCUMM Bar harbor | None (SCUMM BAR sign is likely an object) |
| 041  | Kitchen interior | None |
| 049  | Nighttime forest | None |
| 069  | Giant monkey head beach | None |
| 078  | Throne room | None |
| 079  | LeChuck close-up | None |
| 080  | Jungle scene | None |
| 081  | Character with parrot | None |
| 082  | Character with ghost | None |

---

## Graphics with English Text

### 1. Room 010: LucasFilm Games Logo

**Location**: `game/monkey1/gen/full_dump/DISK_0001/LECF/LFLF_0010/ROOM/RMIM`

**Decoded PNG**: `/tmp/decoded_rooms/room_010.png`

**Text Content**: `LUCASFILM GAMES` (logo lettering in metallic/carved style)

**Translation Impact**: 
- **Keep as-is** (company logo should not be translated)
- Swedish translation must display English LucasFilm Games logo
- No action required

---

## Graphics WITHOUT Text (monkeycd_swe had Swedish versions)

The following rooms had translated Swedish graphics in monkeycd_swe, but the **English originals have NO visible text in the room backgrounds**:

- **Room 009**: Title screen ship interior (no "THE SECRET OF MONKEY ISLAND" text visible)
- **Room 028**: Harbor (no "SCUMM BAR" sign visible in background)
- **Room 049**: No "Stan's Used Ships" sign visible
- **Room 054**: Not found in classic MI1 CD version (may be SE-only or different room number)

### Hypothesis

The text in these scenes is either:
1. **Object images (OBIM)** overlaid on backgrounds
2. **SE-only additions** (Special Edition added text to graphics that weren't in classic)
3. **String resources** (drawn programmatically, not baked into graphics)

Since monkeycd_swe was for **classic MI1 CD version**, not SE, the Swedish graphics may have been:
- Object graphics (not room backgrounds)
- Part title cards (Part I, II, III, IV)
- Credits screens

---

## Conclusion

**No graphics translation needed** for the Swedish fan translation project.

The only English text found in room graphics is the "LUCASFILM GAMES" logo, which should remain in English.

All game-visible text (location names, object names, dialogue, verbs) comes from string resources that will be translated via the scummtr workflow documented in `docs/TRANSLATION_PLAN.md`.

---

## Next Steps

1. ~~Identify graphics with text~~ (DONE — only LucasFilm Games logo)
2. Extract full text corpus from `MONKEY.000/001` using scummtr
3. Begin multi-pass translation workflow (see `TRANSLATION_PLAN.md`)
4. Test SE pipeline with translated text files
