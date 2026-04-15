# Dynamic Name Replacements

Object and actor names that change at runtime via `setObjectName` (0x54/0xD4)
and `ActorOps Name` (0x13/0x93). The SE writes object names **in-place** into
the OBNA buffer with no bounds check — padding is required to prevent overflow.

Source: `translation/monkey1/dynamic_names.json` (gitignored, built by
`scripts/extract_assets.sh`). Padding applied automatically by `scripts/build.sh`.

---

## Objects

| Obj | OBNA (buffer) | Buf | Longest replacement | Len | Fit |
|-----|---------------|-----|---------------------|-----|-----|
| 0058 | `dry river bed` | 13 | `river` | 5 | 8 spare |
| 0066 | `dry river bed` | 13 | `river` | 5 | 8 spare |
| 0091 | `giant piece of rope@@@@@@@@@@@@@...` | 41 | `infinitesimally small rope` | 26 | 15 spare |
| 0157 | `small key` | 9 | `small key` | 9 | EXACT |
| 0169 | `rock on top of note` | 19 | `noteworthy rock` | 15 | 4 spare |
| 0263 | `rowboat@@@@@@@@@` | 16 | `rowboat and oars` | 16 | EXACT |
| 0272 | `memo@@@@@@@@@@@@@@@@` | 20 | `a whole lot of memos` | 20 | EXACT |
| 0294 | `necklace on navigator` | 21 | `necklace on navigator` | 21 | EXACT |
| 0309 | `loose board` | 11 | `loose board` | 11 | EXACT |
| 0322 | `important-looking pirates` | 25 | `cook` | 4 | 21 spare |
| 0377 | `chicken@@@@@@@@` | 15 | `rubber chicken` | 14 | 1 spare |
| 0405 | `prisoner` | 8 | `Otis` | 4 | 4 spare |
| 0420 | `cake@@@@@` | 9 | `file` | 4 | 5 spare |
| 0467 | `deadly piranha poodles@@@@` | 26 | `deadly piranha poodles@@@@` | 26 | EXACT |
| 0478 | `door@@@@@@@@@@@@@@@@@@` | 22 | `murderous winged devil` | 22 | EXACT |
| 0488 | `@@@@@ pieces of eight@@` | 23 | `1 piece of eight` | 16 | 7 spare |
| 0566 | `hunk of meat@@@@@@@@@@` | 22 | `meat with condiment` | 19 | 3 spare |
| 0568 | `fish@@@@@@@@@@@@@@@` | 19 | `fish with condiment` | 19 | EXACT |
| 0574 | `pot o' stew@@@@@` | 16 | `meat in stew` | 12 | 4 spare |
| 0641 | `Manual of Style@@` | 17 | `stylish confetti` | 16 | 1 spare |
| 0646 | `tremendous yak@@@@@@@@@@@@@@@@@@` | 32 | `tremendous dangerous-looking yak` | 32 | EXACT |
| 0648 | `quarrelsome rhinoceros` | 22 | `rhinoceros toenails` | 19 | 3 spare |
| 0649 | `gopher@@@@@@@@@@` | 16 | `funny little man` | 16 | EXACT |
| 0650 | `shredder` | 8 | `shredder` | 8 | EXACT |
| 0694 | `boat@@@@@@@@@@` | 14 | `The Sea Monkey` | 14 | EXACT |
| 0823 | `voodoo root@@@@@@@@@@@@@@@@` | 27 | `magic seltzer bottle` | 20 | 7 spare |
| 0840 | `door@@@@@@@@@@@@@@@@@` | 21 | `squeaky door@@@` | 15 | 6 spare |
| 0882 | `spyglass` | 8 | `lens` | 4 | 4 spare |
| 0912 | `clearing` | 8 | `circus` | 6 | 2 spare |
| 0915 | `lights@@@@@@@@@@@@@` | 19 | `Used Ship Emporium` | 18 | 1 spare |

### All replacement names by object

**#0058** `dry river bed`
- `river` (5) — R015 SCRP:SCRP_0044
- `river` (5) — R004 ENCD:ENCD

**#0066** `dry river bed`
- `river` (5) — R005 ENCD:ENCD

**#0091** `giant piece of rope@@@@@@@@@@@@@@@@@@@@@@`
- `piece of rope` (13) — R008 VERB:OBCD_0091
- `small piece of rope` (19) — R008 VERB:OBCD_0091
- `tiny piece of rope` (18) — R008 VERB:OBCD_0091
- `dinky little rope` (17) — R008 VERB:OBCD_0091
- `infinitesimally small rope` (26) — R008 VERB:OBCD_0091

**#0157** `small key`
- `prize` (5) — R014 SCRP:SCRP_0185
- `small key` (9) — R014 VERB:OBCD_0157

**#0169** `rock on top of note`
- `flint` (5) — R020 SCRP:SCRP_0167
- `noteworthy rock` (15) — R020 SCRP:SCRP_0167
- `flint` (5) — R015 VERB:OBCD_0169

**#0263** `rowboat@@@@@@@@@`
- `rowboat and oars` (16) — R020 LSCR:LSCR_0200
- `rowboat and oars` (16) — R020 ENCD:ENCD

**#0272** `memo@@@@@@@@@@@@@@@@`
- `memos` (5) — R020 SCRP:SCRP_0166
- `a few memos` (11) — R020 SCRP:SCRP_0166
- `several memos` (13) — R020 SCRP:SCRP_0166
- `a bunch of memos` (16) — R020 SCRP:SCRP_0166
- `a pile of memos` (15) — R020 SCRP:SCRP_0166
- `a whole lot of memos` (20) — R020 SCRP:SCRP_0166
- `too many memos` (14) — R020 SCRP:SCRP_0166

**#0294** `necklace on navigator`
- `necklace on navigator` (21) — R025 SCRP:SCRP_0110
- `necklace on Guybrush` (20) — R025 SCRP:SCRP_0141
- `necklace on navigator` (21) — R025 VERB:OBCD_0294
- `necklace on navigator` (21) — R025 VERB:OBCD_0294

**#0309** `loose board`
- `hole` (4) — R027 ENCD:ENCD
- `hole` (4) — R027 VERB:OBCD_0309
- `loose board` (11) — R027 VERB:OBCD_0309

**#0322** `important-looking pirates`
- `cook` (4) — R028 LSCR:LSCR_0200
- `cook` (4) — R028 ENCD:ENCD

**#0377** `chicken@@@@@@@@`
- `rubber chicken` (14) — R029 VERB:OBCD_0377

**#0405** `prisoner`
- `Otis` (4) — R031 LSCR:LSCR_0202

**#0420** `cake@@@@@`
- `file` (4) — R031 VERB:OBCD_0420

**#0467** `deadly piranha poodles@@@@`
- `sleeping piranha poodles` (24) — R036 LSCR:LSCR_0201
- `deadly piranha poodles@@@@` (26) — R036 LSCR:LSCR_0202
- `sleeping piranha poodles` (24) — R036 ENCD:ENCD

**#0478** `door@@@@@@@@@@@@@@@@@@`
- `murderous winged devil` (22) — R037 SCRP:SCRP_0049
- `door` (4) — R037 LSCR:LSCR_0201

**#0488** `@@@@@ pieces of eight@@`
- `1 piece of eight` (16) — R038 VERB:OBCD_0488

**#0566** `hunk of meat@@@@@@@@@@`
- `meat with condiment` (19) — R041 SCRP:SCRP_0182
- `stewed meat` (11) — R041 LSCR:LSCR_0214

**#0568** `fish@@@@@@@@@@@@@@@`
- `stewed fish` (11) — R041 LSCR:LSCR_0214
- `fish with condiment` (19) — R041 VERB:OBCD_0568

**#0574** `pot o' stew@@@@@`
- `spicy stew` (10) — R041 LSCR:LSCR_0213
- `meat in stew` (12) — R041 LSCR:LSCR_0213
- `fish in stew` (12) — R041 LSCR:LSCR_0213
- `spicy stew` (10) — R041 LSCR:LSCR_0214
- `pot o' stew` (11) — R041 LSCR:LSCR_0214

**#0641** `Manual of Style@@`
- `stylish confetti` (16) — R053 LSCR:LSCR_0211

**#0646** `tremendous yak@@@@@@@@@@@@@@@@@@`
- `tremendous dangerous-looking yak` (32) — R053 LSCR:LSCR_0210

**#0648** `quarrelsome rhinoceros`
- `rhinoceros toenails` (19) — R053 LSCR:LSCR_0211
- `lock@@@` (7) — R053 LSCR:LSCR_0211

**#0649** `gopher@@@@@@@@@@`
- `another gopher` (14) — R053 LSCR:LSCR_0210
- `gopher horde` (12) — R053 LSCR:LSCR_0210
- `funny little man` (16) — R053 LSCR:LSCR_0210
- `lock` (4) — R053 LSCR:LSCR_0210

**#0650** `shredder`
- `shredder` (8) — R053 LSCR:LSCR_0211
- `fire` (4) — R053 LSCR:LSCR_0211

**#0694** `boat@@@@@@@@@@`
- `The Sea Monkey` (14) — R059 SCRP:SCRP_0056

**#0823** `voodoo root@@@@@@@@@@@@@@@@`
- `seltzer` (7) — R010 SCRP:SCRP_0001
- `seltzer` (7) — R010 SCRP:SCRP_0001
- `seltzer` (7) — R010 SCRP:SCRP_0001
- `magic seltzer bottle` (20) — R025 SCRP:SCRP_0106

**#0840** `door@@@@@@@@@@@@@@@@@`
- `door` (4) — R077 ENCD:ENCD
- `squeaky door` (12) — R077 ENCD:ENCD
- `door` (4) — R077 ENCD:ENCD
- `door` (4) — R074 VERB:OBCD_0815
- `squeaky door@@@` (15) — R077 VERB:OBCD_0840

**#0882** `spyglass`
- `lens` (4) — R080 VERB:OBCD_0882

**#0912** `clearing`
- `circus` (6) — R085 ENCD:ENCD

**#0915** `lights@@@@@@@@@@@@@`
- `Used Ship Emporium` (18) — R085 ENCD:ENCD

---

## Actors

**Actor 1**: `Guybrush`
**Actor 2**: `lookout`, `monkey`
**Actor 3**: `Citizen of M\x88l\x82e`, `Fettucini Brothers`, `important-looking pirates`, `native`
**Actor 4**: `Fettucini Brothers`, `Otis`, `native`, `prisoner`
**Actor 5**: `native`, `troll`
**Actor 7**: `Herman Toothrot`
**Actor 11**: `storekeeper`

---

## Variable-target (15)

Target determined at runtime — cannot resolve statically.
Mug states (Local[0]/1) target objects #362–#366 (buffer 16–17, fits).

- `melting mug` (11) → Local[0] (R041)
- `mug near death` (14) → Local[0] (R041)
- `pewter wad` (10) → Local[0] (R041)
- `mug o' grog` (11) → Local[1] (R041)
- `pewter wad` (10) → Local[0] (R041)
- `mug o' grog` (11) → Local[0] (R041)
