# Translation Plan — The Secret of Monkey Island SE
## Swedish Translation Samples

**Version:** 1.0  
**Last updated:** 2026-04-09  
**Scope:** `translation/monkey1/swedish.txt` — **sample translations** for demonstration and learning purposes

---

## Overview

**IMPORTANT: This is a SAMPLE translation project for educational/demonstration purposes only. We are NOT translating the complete game, only selected representative rooms and passages to demonstrate translation methodology.**

This document describes the multi-pass workflow for producing Swedish translation samples. The approach follows how professional game localizers work today: functional fidelity over literal fidelity, preserving humor and tone, with special attention to language-dependent jokes.

**Sample Selection Strategy:**
- Representative rooms from different game areas
- Key dialogue scenes showing character voices
- Examples of different content types (objects, dialogue, puzzles, humor)
- Challenging passages requiring creative translation

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

## Translation Workflow

**Simple Command-Based Approach:**

1. **To start/continue:** User says "translate the next room" or "continue Pass 2"
2. **Claude will:**
   - Check the current pass's room list
   - Find the first uncompleted room (unmarked checkbox)
   - Translate all strings for that room
   - Update the checkbox to [x] in this plan document
   - Add translations to `swedish.txt` at the correct line numbers
3. **User reviews** the translation for that room
4. **User says** "translate the next room" to continue, or provides feedback for revisions

**To specify a room:** "translate room 007" or "do room 010 next"

**To skip ahead:** "start Pass 3" (moves to next pass section)

**Progress tracking:** Each pass has a checklist. Check this file to see what's done.

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

### Pass 2 — Sample Translation (Selected Rooms)
**Goal:** Translate selected representative rooms as samples, one room at a time.  
**Input:** English strings for each room, `glossary.md`, `pun_inventory.md`  
**Output:** Swedish translation samples added to `translation/monkey1/swedish.txt`

**Workflow:** User says "translate the next room" and Claude translates the next uncompleted room from this list.

**Legend:**
- [E] ready = English text with [E] prefix already added (shows in git diff)

**Purpose:**
This is a complete reference list for planning translation work. NOT a commitment to translate all content — this is a planning document for future translators to track progress systematically.

---

### Pass 3 — Consistency Review
**Goal:** Verify every object name, character name, and recurring phrase is used consistently throughout the entire file.  
**Input:** Complete `translation/monkey1/swedish.txt`, `glossary.md`  
**Output:** Corrections applied in-place; `glossary.md` updated with any new decisions

Check specifically:
- Every `OBNA` entry vs. how the object is referred to in nearby `VERB` entries
- Character names in dialogue vs. in stage directions vs. in object names
- Verb UI strings (e.g. "Öppna" for Open) used consistently
- The insult swordfighting comebacks still match their insults after other changes

---

### Pass 4 — Pun and Wordplay Polish
**Goal:** Review all flagged entries from `pun_inventory.md` in context, and improve any translations that feel awkward.  
**Input:** `pun_inventory.md`, current `translation/monkey1/swedish.txt`  
**Output:** Corrections applied in-place; `pun_inventory.md` annotated with resolution status

For each flagged string:
1. Read the Swedish equivalent in context (read adjacent strings in the same room/script).
2. Does it land as a joke in Swedish? Does the comedy beat survive?
3. If not, revise. Document what changed.

---

### Pass 5 — Length Validation
**Goal:** Ensure no string exceeds 256 characters (SE hard limit).  
**Input:** `translation/monkey1/swedish.txt`  
**Output:** List of violations; shortened strings applied in-place

Flag any line where the text portion (after the `]`) exceeds 256 characters including SCUMM escape codes. Shorten by:
1. Rephrasing (preferred — preserve meaning)
2. Breaking into two display segments using `\255\003` (only if the original also uses a pause)
3. Last resort: cutting content (document what was removed)

---

### Pass 6 — Final Read-Through
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
