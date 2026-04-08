# Translation Plan — The Secret of Monkey Island SE
## Swedish Fan Translation

**Version:** 1.0  
**Last updated:** 2026-04-08  
**Scope:** `translation/monkey1/swedish.txt` — full fresh translation from English

---

## Overview

This document describes the multi-pass workflow Claude uses to produce the Swedish translation. The previous `monkeycd_swe` Swedish text is retired as source material; we produce a new translation from the extracted English source.

The approach follows how professional game localizers work today: functional fidelity over literal fidelity, preserving humor and tone, with special attention to language-dependent jokes.

**For technical details on file format, opcodes, and control codes, see `TRANSLATION_GUIDE.md`.**

---

## Translation Philosophy

### General Approach

- **Functional equivalence, not word-for-word.** The Swedish text should feel natural and funny to a native Swedish speaker, not like translated English.
- **Match register and tone.** Guybrush is a young, naive, occasionally sarcastic would-be pirate. NPCs have distinct personalities. Preserve this.
- **Length awareness.** Swedish is on average 10–20% longer than English. The hard limit per string is **256 characters** (SE fixed-stride format).

### Proper Nouns

| Category | Decision |
|----------|----------|
| Character names | **Keep** (Guybrush Threepwood, Elaine Marley, LeChuck) |
| Place names | **Keep** (Mêlée Island, Monkey Island, Scabb Island) |
| Item names | **Translate** (swords, ropes, keys, grog mugs) |
| Business names | **Keep** (The Scumm Bar, Stan's Previously Owned Vessels) |
| Fictional items | **Translate playfully** (maintain absurdity) |

### Humor

MI1 is a comedy game built on absurdist humor, pirate tropes, insult swordfighting, and fourth-wall breaks.

**Always prioritize the joke over the literal meaning.** If a joke only works in English, create an equivalent Swedish joke.

---

## Passes

### Pass 0 — Preprocessing: Glossary and Pun Inventory
**Goal:** Create the consistency foundation before any translation begins.  
**Input:** `game/monkey1/gen/strings/english.txt`  
**Output:** `translation/monkey1/glossary.md`, `translation/monkey1/pun_inventory.md`

Steps:
1. Scan all strings and extract every proper noun (character names, place names, item names).
2. For each item name that appears in both `OBNA` and `VERB` strings, note both occurrences — they must match.
3. Flag every string that contains: wordplay, puns, rhymes, alliteration, idioms, jokes that rely on English homophones or double meanings. Write these to `pun_inventory.md` with: the English string, what makes it language-dependent, and a proposed Swedish equivalent.
4. Identify the insult swordfighting strings — these are a complete sub-system where insults must have matching comebacks. Handle as a block.
5. Write `glossary.md` with all translation decisions for proper nouns and recurring terms.

**This pass must complete before any translation pass starts.**

---

### Pass 1 — Insult Swordfighting (standalone sub-system)
**Goal:** Translate the insult/comeback pairs as a coherent, funny system.  
**Input:** Insult swordfighting strings (flagged in Pass 0), `glossary.md`  
**Output:** Translated insult/comeback pairs in `translation/monkey1/swedish.txt` (just this section)

Note from user: Keep in mind that in the sword fight with the Sword master different insults are used and the player needs to select from the earlier come backs. I.e. these special swoprd master insults must match one of the existing come backs. If they are too similar to the original insults it becomes too easy, but if the come back doesn't make sense then it becomes too hard. 
The insult swordfighting system has paired strings: each insult has exactly one correct comeback. Both sides must be funny and the comeback must logically respond to the insult. This requires treating them as a creative writing task, not a translation task. Create Swedish wordplay that works, even if it diverges significantly from the English.


---

### Pass 2 — Initial Translation: Act 1 (The Three Trials, Mêlée Island part 1)
**Goal:** Translate all strings from rooms 001–025.  
**Input:** English strings for rooms 001–025, `glossary.md`, `pun_inventory.md`  
**Output:** Swedish strings appended to `translation/monkey1/swedish.txt`

Covers: Lookout cliff, jungle, SCUMM Bar, shops, jail, Governor's mansion first visit.

---

### Pass 3 — Initial Translation: Act 2 (The Three Trials, Mêlée Island part 2)
**Goal:** Translate all strings from rooms 026–050.  
**Input:** English strings for rooms 026–050, `glossary.md`, `pun_inventory.md`  
**Output:** Swedish strings appended to `translation/monkey1/swedish.txt`

Covers: Sword training, store of phony goods, circus tent, church, Stan's ship lot, Carla/Otis/Meathook recruitment.

---

### Pass 4 — Initial Translation: Act 3 (The Journey / Ship)
**Goal:** Translate all strings from rooms 051–070.  
**Input:** English strings for rooms 051–070, `glossary.md`, `pun_inventory.md`  
**Output:** Swedish strings appended to `translation/monkey1/swedish.txt`

Covers: The ship, crew quarters, ocean, arrival at Monkey Island.

---

### Pass 5 — Initial Translation: Act 4 (Under Monkey Island)
**Goal:** Translate all strings from rooms 071–099.  
**Input:** English strings for rooms 071–099, `glossary.md`, `pun_inventory.md`  
**Output:** Swedish strings appended to `translation/monkey1/swedish.txt`

Covers: Monkey Island jungle, Stan's ghost ship, cannibal village, Herman Toothrot, underground.

---

### Pass 6 — Initial Translation: Act 5 and global scripts
**Goal:** Translate all remaining strings (rooms 100+, global SCRP resources).  
**Input:** English strings for rooms 100+, `glossary.md`, `pun_inventory.md`  
**Output:** Swedish strings appended to `translation/monkey1/swedish.txt`

Covers: LeChuck's fortress, finale, all global cutscene scripts.

---

### Pass 7 — Consistency Review
**Goal:** Verify every object name, character name, and recurring phrase is used consistently throughout the entire file.  
**Input:** Complete `translation/monkey1/swedish.txt`, `glossary.md`  
**Output:** Corrections applied in-place; `glossary.md` updated with any new decisions

Check specifically:
- Every `OBNA` entry vs. how the object is referred to in nearby `VERB` entries
- Character names in dialogue vs. in stage directions vs. in object names
- Verb UI strings (e.g. "Öppna" for Open) used consistently
- The insult swordfighting comebacks still match their insults after other changes

---

### Pass 8 — Pun and Wordplay Polish
**Goal:** Review all flagged entries from `pun_inventory.md` in context, and improve any translations that feel awkward.  
**Input:** `pun_inventory.md`, current `translation/monkey1/swedish.txt`  
**Output:** Corrections applied in-place; `pun_inventory.md` annotated with resolution status

For each flagged string:
1. Read the Swedish equivalent in context (read adjacent strings in the same room/script).
2. Does it land as a joke in Swedish? Does the comedy beat survive?
3. If not, revise. Document what changed.

---

### Pass 9 — Length Validation
**Goal:** Ensure no string exceeds 256 characters (SE hard limit).  
**Input:** `translation/monkey1/swedish.txt`  
**Output:** List of violations; shortened strings applied in-place

Flag any line where the text portion (after the `]`) exceeds 256 characters including SCUMM escape codes. Shorten by:
1. Rephrasing (preferred — preserve meaning)
2. Breaking into two display segments using `\255\003` (only if the original also uses a pause)
3. Last resort: cutting content (document what was removed)

---

### Pass 10 — Final Read-Through
**Goal:** Read the whole translation as a playthrough, room by room. Catch anything that sounds robotic, inconsistent, or unfunny.  
**Input:** Complete `translation/monkey1/swedish.txt`  
**Output:** Final corrections applied in-place

Read room by room in order. For each room, read all its strings together as a sequence — this simulates how a player experiences them. Fix anything that reads awkwardly in sequence even if each individual line looked fine in isolation.

---

## File Layout

```
translation/monkey1/
  swedish.txt          — The translation file (scummtr format, UTF-8)
  glossary.md          — Proper nouns, recurring terms, translation decisions
  pun_inventory.md     — Language-dependent strings: English original + Swedish solution
```

The `swedish.txt` file is the only file that feeds into the build pipeline. The other two are working documents.

---

## Quality Checklist

Before committing any pass:
- [ ] Control codes preserved (see `TRANSLATION_GUIDE.md`)
- [ ] No line exceeds 256 characters
- [ ] Object names match their references in dialogue
- [ ] Glossary consulted for all proper nouns
- [ ] No accidental English (except proper nouns)

---

## Notes on Specific Challenges

### "Monkey Island" — the island name
Keep as "Monkey Island" (known in Swedish as "Apenöarna" colloquially, but the game's own title is the established name). The `\015` in game-internal references to the island name is a registered-trademark glyph rendered by the engine — preserve it.

### The SCUMM Bar
Keep "SCUMM Bar" — it's a meta-joke referencing the engine. A Swedish player who knows adventure games will appreciate it unchanged.

### Voodoo and supernatural elements
"Voodoo Lady", "Root Beer", "Ghost Pirates" — keep supernatural item names in Swedish where they're descriptive ("spökpirater" etc.), keep proper nouns in English.

### Grog
"Grog" has no Swedish equivalent. Keep "Grog" but translate descriptions around it.

### Text speed
Note for testers: Swedish text is longer than English. Set ScummVM text speed to 120+ or use the SE's auto-advance feature. The SE's timed text display may cut off longer strings — test in-game.
