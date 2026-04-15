# Dynamic Object and Actor Name Replacements

This document maps every runtime name change in MI1 — set by game scripts via
`setObjectName` (opcodes 0x54/0xD4) and `ActorOps Name` (opcodes 0x13/0x93).

Source: decompiled with `descumm -5` from scummvm-tools, applied to all SCRP,
LSCR, ENCD, EXCD, and VERB script blocks extracted from the original MONKEY1.001.

The `@` character is invisible in SCUMM rendering (zero width, zero pixels).
LucasArts used it to pad object name buffers to the maximum length they would
ever need at runtime.

---

## Part 1: Object Name Changes (`setObjectName`)

### #0058 — `dry river bed` → `river`

| Replacement | Source |
|---|---|
| `river` | Room 004 ENCD, Room 015 SCRP#0044 |

### #0066 — `dry river bed` → `river`

| Replacement | Source |
|---|---|
| `river` | Room 005 ENCD |

### #0091 — `giant piece of rope@@@@@@@@@@@@@@@@@@@@@@` (41 bytes)

The rope shrinks each time you cut/use it. `VAR_ME` means the object that owns the VERB script.

| Replacement | Len | Source |
|---|---|---|
| `piece of rope` | 13 | Room 008 VERB#0091 |
| `small piece of rope` | 19 | Room 008 VERB#0091 |
| `tiny piece of rope` | 18 | Room 008 VERB#0091 |
| `dinky little rope` | 17 | Room 008 VERB#0091 |
| `infinitesimally small rope` | 26 | Room 008 VERB#0091 |

Buffer: 41. Longest: 26. **15 spare.**

### #0157 — `small key`

| Replacement | Source |
|---|---|
| `small key` | Room 014 VERB#0157 |
| `prize` | Room 014 SCRP#0185 |

### #0169 — `rock on top of note`

| Replacement | Source |
|---|---|
| `flint` | Room 015 VERB#0169, Room 020 SCRP#0167 |
| `noteworthy rock` | Room 020 SCRP#0167 |

### #0263 — `rowboat@@@@@@@@@` (16 bytes)

| Replacement | Len | Source |
|---|---|---|
| `rowboat and oars` | 16 | Room 020 ENCD, Room 020 LSCR#200 |

Buffer: 16. Longest: 16. **EXACT FIT.**

### #0272 — `memo@@@@@@@@@@@@@@@@` (20 bytes)

Memo count increases as you pick them up.

| Replacement | Len | Source |
|---|---|---|
| `memos` | 5 | Room 020 SCRP#0166 |
| `a few memos` | 11 | Room 020 SCRP#0166 |
| `several memos` | 13 | Room 020 SCRP#0166 |
| `a bunch of memos` | 16 | Room 020 SCRP#0166 |
| `a pile of memos` | 15 | Room 020 SCRP#0166 |
| `a whole lot of memos` | 20 | Room 020 SCRP#0166 |
| `too many memos` | 14 | Room 020 SCRP#0166 |

Buffer: 20. Longest: 20. **EXACT FIT.**

### #0294 — `necklace on navigator`

| Replacement | Source |
|---|---|
| `necklace on navigator` | Room 025 VERB#0294, Room 025 SCRP#0110 |
| `necklace on Guybrush` | Room 025 SCRP#0141 |

### #0309 — `loose board`

| Replacement | Source |
|---|---|
| `hole` | Room 027 VERB#0309, Room 027 ENCD |
| `loose board` | Room 027 VERB#0309 (reset) |

### #0322 — `important-looking pirates`

| Replacement | Source |
|---|---|
| `cook` | Room 028 ENCD, Room 028 LSCR#200 |

### #0362–#0366 — `mug@@@@@@@@@@@@@@` (17 bytes, #0365 is 16)

Grog dissolves the mug. Target is `Local[0]`/`Local[1]` (variable — the mug the player is holding).

| Replacement | Len | Source |
|---|---|---|
| `mug o' grog` | 11 | Room 041 SCRP#0069, Room 041 LSCR#215 |
| `melting mug` | 11 | Room 041 SCRP#0068 |
| `mug near death` | 14 | Room 041 SCRP#0068 |
| `pewter wad` | 10 | Room 041 SCRP#0068, Room 041 SCRP#0069 |

Buffer: 17 (16 for #0365). Longest: 14. **3 spare (2 for #0365).**

### #0377 — `chicken@@@@@@@@` (15 bytes)

| Replacement | Len | Source |
|---|---|---|
| `rubber chicken` | 14 | Room 029 VERB#0377 |

Buffer: 15. Longest: 14. **1 spare.**

### #0405 — `prisoner`

| Replacement | Source |
|---|---|
| `Otis` | Room 031 LSCR#202 |

### #0420 — `cake@@@@@` (9 bytes)

| Replacement | Len | Source |
|---|---|---|
| `file` | 4 | Room 031 VERB#0420 |

Buffer: 9. Longest: 4. **5 spare.**

### #0467 — `deadly piranha poodles@@@@` (26 bytes)

| Replacement | Len | Source |
|---|---|---|
| `sleeping piranha poodles` | 24 | Room 036 ENCD, Room 036 LSCR#201 |
| `deadly piranha poodles@@@@` | 26 | Room 036 LSCR#202 (padded in script too) |

Buffer: 26. Longest: 26. **EXACT FIT.**

### #0478 — `door@@@@@@@@@@@@@@@@@@` (22 bytes)

| Replacement | Len | Source |
|---|---|---|
| `door` | 4 | Room 037 LSCR#201 |
| `murderous winged devil` | 22 | Room 037 SCRP#0049 |

Buffer: 22. Longest: 22. **EXACT FIT.**

### #0488 — `@@@@@ pieces of eight@@` (23 bytes)

| Replacement | Len | Source |
|---|---|---|
| `1 piece of eight` | 16 | Room 038 VERB#0488 |
| `getInt(Var[195]) + " pieces of eight"` | ≤23 | Room 038 VERB#0488 |

The `Var[195]` is the gold count. Buffer: 23.

### #0566 — `hunk of meat@@@@@@@@@@` (22 bytes)

| Replacement | Len | Source |
|---|---|---|
| `meat with condiment` | 19 | Room 041 SCRP#0182 |
| `stewed meat` | 11 | Room 041 LSCR#214 |

Buffer: 22. Longest: 19. **3 spare.**

### #0568 — `fish@@@@@@@@@@@@@@@` (19 bytes)

| Replacement | Len | Source |
|---|---|---|
| `fish with condiment` | 19 | Room 041 VERB#0568 |
| `stewed fish` | 11 | Room 041 LSCR#214 |

Buffer: 19. Longest: 19. **EXACT FIT.**

### #0574 — `pot o' stew@@@@@` (16 bytes)

| Replacement | Len | Source |
|---|---|---|
| `spicy stew` | 10 | Room 041 LSCR#213, LSCR#214 |
| `meat in stew` | 12 | Room 041 LSCR#213 |
| `fish in stew` | 12 | Room 041 LSCR#213 |
| `pot o' stew` | 11 | Room 041 LSCR#214 |

Buffer: 16. Longest: 12. **4 spare.**

### #0641 — `Manual of Style@@` (17 bytes)

| Replacement | Len | Source |
|---|---|---|
| `stylish confetti` | 16 | Room 053 LSCR#211 |

Buffer: 17. Longest: 16. **1 spare.**

### #0646 — `tremendous yak@@@@@@@@@@@@@@@@@@` (32 bytes)

| Replacement | Len | Source |
|---|---|---|
| `tremendous dangerous-looking yak` | 32 | Room 053 LSCR#210 |

Buffer: 32. Longest: 32. **EXACT FIT.**

### #0648 — `quarrelsome rhinoceros` (22 bytes, no `@` padding)

| Replacement | Len | Source |
|---|---|---|
| `rhinoceros toenails` | 19 | Room 053 LSCR#211 |
| `lock@@@` | 7 | Room 053 LSCR#211 |

### #0649 — `gopher@@@@@@@@@@` (16 bytes)

| Replacement | Len | Source |
|---|---|---|
| `another gopher` | 14 | Room 053 LSCR#210 |
| `gopher horde` | 12 | Room 053 LSCR#210 |
| `funny little man` | 16 | Room 053 LSCR#210 |
| `lock` | 4 | Room 053 LSCR#210 |

Buffer: 16. Longest: 16. **EXACT FIT.**

### #0650 — `shredder` (no `@` padding)

| Replacement | Source |
|---|---|
| `shredder` | Room 053 LSCR#211 |
| `fire` | Room 053 LSCR#211 |

### #0694 — `boat@@@@@@@@@@` (14 bytes)

| Replacement | Len | Source |
|---|---|---|
| `The Sea Monkey` | 14 | Room 059 SCRP#0056 |

Buffer: 14. Longest: 14. **EXACT FIT.**

### #0749 — `X@@@` (4 bytes)

No `setObjectName` calls found. Buffer: 4.

### #0799 — `key@@@@@@@` (10 bytes)

No `setObjectName` calls found. Buffer: 10. **7 spare.**

### #0815 — `ghost door` (no `@` padding, but sets #0840)

| Replacement | Source |
|---|---|
| setObjectName(840, `"door"`) | Room 074 VERB#0815 |

### #0823 — `voodoo root@@@@@@@@@@@@@@@@` (27 bytes)

Transforms into seltzer bottle.

| Replacement | Len | Source |
|---|---|---|
| `seltzer` | 7 | Room 010 SCRP#0001 (×3) |
| `magic seltzer bottle` | 20 | Room 025 SCRP#0106 |

Buffer: 27. Longest: 20. **7 spare.**

### #0840 — `door@@@@@@@@@@@@@@@@@` (21 bytes)

Ghost ship door that creaks.

| Replacement | Len | Source |
|---|---|---|
| `door` | 4 | Room 074 VERB#0815, Room 077 ENCD (×2) |
| `squeaky door` | 12 | Room 077 ENCD |
| `squeaky door@@@` | 15 | Room 077 VERB#0840 |

Buffer: 21. Longest: 15. **6 spare.**

### #0882 — `spyglass` (no `@` padding)

| Replacement | Source |
|---|---|
| `lens` | Room 080 VERB#0882 |

### #0912 — `clearing` (no `@` padding)

| Replacement | Source |
|---|---|
| `circus` | Room 085 ENCD |

### #0915 — `lights@@@@@@@@@@@@@` (19 bytes)

| Replacement | Len | Source |
|---|---|---|
| `circus` | 6 | Room 085 ENCD |
| `Used Ship Emporium` | 18 | Room 085 ENCD |

Buffer: 19. Longest: 18. **1 spare.**

---

## Part 2: Actor Name Changes (`ActorOps Name`)

These set character display names dynamically. Actor names appear in dialog
subtitles when the character speaks.

### Actor #1 — Guybrush

| Name | Source |
|---|---|
| `Guybrush` | Room 010 SCRP#0001 |

### Actor #2 — various

| Name | Source |
|---|---|
| `monkey` | Room 020 LSCR#202 |
| `lookout` | Room 038 LSCR#200 |

### Actor #3 — various

| Name | Source |
|---|---|
| `native` | Room 025 LSCR#200 |
| `important-looking pirates` | Room 028 LSCR#204 |
| `Fettucini Brothers` | Room 051 LSCR#204 |
| `Citizen of Mêlée` | Room 035 LSCR#200 |

### Actor #4 — various

| Name | Source |
|---|---|
| `native` | Room 025 LSCR#200 |
| `Otis` | Room 031 LSCR#201 |
| `prisoner` | Room 031 LSCR#201 |
| `Fettucini Brothers` | Room 051 LSCR#204 |

### Actor #5 — various

| Name | Source |
|---|---|
| `native` | Room 025 LSCR#200 |
| `troll` | Room 057 LSCR#200 |

### Actor #7 — Herman Toothrot

| Name | Source |
|---|---|
| `Herman Toothrot` | Rooms 001, 011, 012, 025, 040, 070, 080 (LSCR) |

### Actor #11 — storekeeper

| Name | Source |
|---|---|
| `storekeeper` | Room 030 SCRP#0067 |

### Actor Local[0] — vulture/birds (Room 002)

| Name | Source |
|---|---|
| `@@@@@@@` | Room 002 SCRP#0037, SCRP#0038, SCRP#0039 (init padding) |
| `vulture` | Room 002 SCRP#0037 |
| ` ` (space) | Room 002 SCRP#0035 |

### Local[4] — sword-fighting pirates (Room 085)

| Name | Source |
|---|---|
| `Dirty Rotten Pirate` | Room 085 LSCR#202 |
| `Stinking Pirate` | Room 085 LSCR#202 |
| `Bloodthirsty Pirate` | Room 085 LSCR#202 |
| `Ugly Pirate` | Room 085 LSCR#202 |

---

## Summary

| `@`-padded objects | 26 |
|---|---|
| Total `setObjectName` calls | ~70 |
| Total `ActorOps Name` calls | ~30 |
| EXACT FIT (buffer = longest replacement) | 8 objects |
| Objects with no replacement found | 3 (#0104, #0749, #0799) |
