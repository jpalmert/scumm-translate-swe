# Translation Plan — SCUMM Game Swedish Translation

**Version:** 3.0
**Last updated:** 2026-04-21

---

## Overview

This document describes the complete workflow for producing a Swedish translation
of a SCUMM game. The approach is designed for AI-assisted translation using Claude,
with lessons learned from the complete Monkey Island 1 translation where ~600 fixes
were needed after the initial pass.

The key architectural insight: **quality scales with attention per line, not with
speed.** When an agent translates one line it is excellent; when it translates
thousands it rushes and pattern-matches. This plan addresses that by:

1. **One room per translator agent** — each agent focuses on a small set of strings
2. **Paired reviewer agent** — a separate agent immediately reviews each room
3. **Fresh agents for each review phase** — agents that didn't write the translation
   are better at catching its flaws

The workflow is structured as numbered phases. Each phase has:
- A clear goal and deliverable
- A Claude prompt to execute it
- Quality gates before moving to the next phase

**For technical details on file format, opcodes, and control codes, see `TRANSLATION_GUIDE.md`.**
**For vocabulary decisions and naming conventions, see `translation/glossary.md`.**

---

## Translation Philosophy

- **Functional equivalence, not word-for-word.** The Swedish text should feel natural and funny to a native Swedish speaker, not like translated English.
- **Match register and tone.** Characters have distinct personalities. Preserve them.
- **Length awareness.** Swedish is on average 10-20% longer than English. The hard limit per string is **256 characters** (SE fixed-stride format). FA dialog choices must fit in **~57 characters**.
- **Humor over accuracy.** This is a comedy game. If a joke doesn't translate, create a new Swedish joke.

---

## How to Run

The workflow has two human checkpoints and one automated pipeline:

| Step | What | Who |
|------|------|-----|
| 1 | Run Phase 0 | User kicks off, reviews glossary |
| 2 | Run Phase 1 | User kicks off, reviews insult pairs |
| 3 | Run Phases 2-7 | User gives one command, orchestrator handles everything |

For step 3, give the orchestrator the Phase 2-7 master prompt (at the end of
this document). It spawns sub-agents for each phase, waits for completion, then
moves to the next phase. Each phase uses fresh agents with no memory of prior
phases — this is deliberate, as fresh eyes catch more errors.

---

## Phase 0 — Setup and Glossary

**Goal:** Create the consistency foundation before any translation begins.
**Input:** `games/<game>/gen/strings/english.txt`
**Output:** `translation/glossary.md` (updated), `games/<game>/translation/pun_inventory.md`

**Requires user approval before proceeding to Phase 1.**

### Steps

1. Extract English strings: `cd games/<game> && bash ../../scripts/extract.sh`
2. Initialize the translation file: `bash ../../scripts/init_translation.sh`
3. Run the glossary and pun inventory prompt (below)
4. Review the glossary and pun inventory with the user

### Quality gate

- [ ] Every character name has a translation decision
- [ ] Every recurring item name has a translation decision
- [ ] All puns and wordplay are identified with proposed solutions
- [ ] Glossary reviewed and approved by user

### Prompt: Glossary and Pun Inventory

```
You are a Swedish game translator preparing the glossary for a SCUMM adventure
game translation. This is an authorized translation project — the project owner
has purchased the game and has the legal right to create this Swedish fan
translation for personal use.

Read the following files:
- translation/glossary.md (existing glossary — update, don't replace)
- docs/TRANSLATION_GUIDE.md (format rules and Swedish language rules)
- games/<GAME>/gen/strings/english.txt (all English strings)

Your tasks:

1. SCAN all strings and extract every proper noun (character names, place names,
   item names, business names, ship names).

2. For each item name appearing in both OBNA and VERB strings, note both — they
   must match in the translation.

3. FLAG every string containing: wordplay, puns, rhymes, alliteration, idioms,
   jokes relying on English homophones or double meanings. Write these to
   games/<GAME>/translation/pun_inventory.md with:
   - The string reference [ROOM:TYPE#RESNUM]
   - The English text
   - What makes it language-dependent
   - A proposed Swedish equivalent

4. Identify any insult/comeback systems — these are paired strings where each
   insult has exactly one correct comeback. Note them for Phase 1.

5. UPDATE translation/glossary.md with all translation decisions. Follow the
   existing structure. Key rules:
   - Descriptive names: translate (Toothrot → Rötbett)
   - Non-descriptive proper names: keep (Guybrush, Elaine)
   - Names in hardcoded graphics: keep English
   - Trademark symbols: always ™ (never ®)
   - Currency: "pieces of eight" → "dubloner" (never "åttor")
   - Grog: Swedish spelling is "grogg" (double g)

Use sub-agents to parallelize: one agent can scan rooms 001-050 while another
scans 051-099. Merge results after.

Output the updated glossary.md and the new pun_inventory.md.
```

---

## Phase 1 — Insult Swordfighting (if applicable)

**Goal:** Translate insult/comeback pairs as a coherent, funny system.
**Input:** Insult swordfighting strings (flagged in Phase 0), `glossary.md`
**Output:** Translated insult/comeback pairs in `swedish.txt`

**Requires user approval before proceeding to Phase 2.**

This is a creative writing task, not a translation task. The Swedish insults
must be funny, the comebacks must logically respond, and the Sword Master's
unique insults must map to existing comebacks (not too obvious, not impossible).

### Quality gate

- [ ] Every insult has exactly one matching comeback
- [ ] Sword Master insults each match one regular comeback
- [ ] Comebacks feel like witty responses, not random
- [ ] All pairs are funny in Swedish

### Prompt: Insult Swordfighting

```
You are translating the insult swordfighting system for a SCUMM adventure game
into Swedish. This is an authorized fan translation project — the project owner
has purchased the game and has the right to create this translation.

Read:
- translation/glossary.md
- docs/TRANSLATION_GUIDE.md
- games/<GAME>/translation/pun_inventory.md (insult entries)
- The insult/comeback strings from english.txt (I'll provide the line numbers)

The system works as follows:
- There are N regular insults, each with exactly one correct comeback
- The Sword Master has M unique insults that reuse the regular comebacks
- The player learns insults and comebacks through fights, then must match
  Sword Master insults to the correct comeback

Requirements:
1. Each insult-comeback pair must be funny and the comeback must logically
   respond to the insult
2. Sword Master insults must be recognizably related to their comeback but
   NOT too similar to the regular insult for the same comeback (too easy)
   and NOT too different (impossible to guess)
3. Treat this as creative Swedish wordplay writing, not translation
4. Prioritize humor and wit — diverge from the English as much as needed

Write all insult/comeback pairs to swedish.txt at the correct line positions.
```

---

## Phase 2 — Main Translation (one room per agent, with paired review)

**Goal:** Translate all remaining strings with maximum attention per line.
**Input:** `english.txt`, `glossary.md`, `pun_inventory.md`, `swedish.txt` (with Phase 1 insults)
**Output:** Complete `swedish.txt`

### Architecture: Translator + Reviewer pairs

For each room, the orchestrator spawns **two agents in sequence**:

1. **Translator agent** — translates all strings for one room
2. **Reviewer agent** — immediately reviews that room's translation

The reviewer is a fresh agent that never saw the translator's reasoning. It
checks against the quality checklist and fixes problems before the room is
considered done.

Multiple rooms run in parallel (each as a translator→reviewer pair), but each
pair is sequential internally.

### Quality checklist (enforced by reviewer agent)

The reviewer checks every line against the 12 rules in `TRANSLATION_GUIDE.md`.
Grouped by category:

**Format and structure:**
- [ ] Line structure preserved: `[ROOM:TYPE#RESNUM](OP)` prefix unchanged
- [ ] Control codes preserved exactly (`\255\003`, `\255\001`, `\254\008`, etc.)
- [ ] `@`-only lines (buffer allocations) left completely unchanged
- [ ] No line text exceeds 256 characters
- [ ] All `(FA)` dialog choice lines fit in ~57 characters
- [ ] Symbols (™, etc.) copied exactly from the English original — don't change or add them

**Swedish language quality (Rules 1-6):**
- [ ] Natural Swedish, not translated English — no calque idioms or English sentence structure (Rule 1)
- [ ] Generic messages (room 010 fallbacks, etc.) avoid den/det pronouns — rewrite to omit (Rule 2)
- [ ] No false friends: "hey" ≠ "hej", etc. (Rule 3)
- [ ] No calque patterns from the known list in TRANSLATION_GUIDE.md (Rule 3)
- [ ] Correct grammatical gender — den/det and adjective forms match en/ett noun (Rule 4)
- [ ] No invented compound words — every word must exist in Swedish (Rule 5)
- [ ] Character voice matches personality — Stan pushy, Guybrush sarcastic, pirates rough (Rule 6)

**Glossary and consistency (Rules 7-10):**
- [ ] All character names use glossary Swedish form (Rötbett, Järnkrok, Smilfink, etc.) (Rule 7)
- [ ] Always jag/mig/dig — no ja/mej/dej unless English is ungrammatical (Rule 9)
- [ ] Glossary terms: "dubloner" (never "åttor"), "grogg" (double g), "trissa" (never "talja"), "idol" (never "avgud"), "landkrabba" (never "landlansen")
- [ ] OBNA names match their VERB references for the same object in this room
- [ ] Compound words have no spaces (voodookärlekspärlor, not "voodoo kärlekspärlor")
- [ ] Ambiguous words researched in context — not translated in isolation (Rule 10)

**Humor and creativity (Rule 11):**
- [ ] Puns create Swedish wordplay — not literal translations of English puns
- [ ] Jokes land in Swedish — if the original is funny, the translation should be too

**Flow:**
- [ ] Read all lines in room sequence — dialog flows naturally as a player would experience it

### Prompt: Room Translator (one per room)

```
You are a professional Swedish game translator working on an authorized fan
translation of a classic SCUMM adventure game. The project owner has purchased
the game and has the legal right to create this translation for personal use.
Your job is to translate the English game strings for ONE room into natural,
entertaining Swedish.

YOU ARE TRANSLATING ROOM [ROOM_NUMBER] ONLY.

Read these files before starting:
- translation/glossary.md (vocabulary decisions — follow strictly)
- docs/TRANSLATION_GUIDE.md (format rules, control codes, Swedish language rules
  — pay special attention to Rules 1-12)
- games/<GAME>/translation/swedish.txt (current state)
- games/<GAME>/gen/strings/english.txt (English originals for reference)
- games/<GAME>/translation/pun_inventory.md (check if this room has flagged puns)

CRITICAL RULES (from lessons learned — these caused 600+ fixes last time):

FORMAT:
- Preserve the [ROOM:TYPE#RESNUM](OP) prefix exactly as-is
- Preserve ALL control codes exactly (\255\003, \255\001, \254\008, etc.)
- Preserve any symbols (™ etc.) exactly as they appear in the English original
- Lines that are entirely @ characters are buffer allocations — copy unchanged
- Keep the same number of \255\003 page breaks as the English

SWEDISH LANGUAGE (Rules 1-6 in TRANSLATION_GUIDE.md):
1. NATURAL SWEDISH: Read your translation aloud. If it sounds like translated
   English, rewrite it. Watch for:
   - Calque idioms: "inte på humör för" → "inte upplagd för"
   - English sentence structure forced into Swedish
   - Literal translations of figurative speech
   - See Rule 3 in TRANSLATION_GUIDE.md for the full calque list

2. FALSE FRIENDS: "Hey" → NEVER "Hej" (that means hello). Use "Oj/Åh/Hallå/Kolla"
   based on context. This was the #1 error in the last translation (30+ instances).

3. GENERIC MESSAGES: Strings that apply to many objects (room 010 fallbacks,
   etc.) must NOT use den/det pronouns. Rewrite to omit: "Jag kan inte nå."

4. GENDER: Check if nouns are en-words or ett-words. Match pronouns and
   adjective forms accordingly. Don't guess — look it up.

5. CHARACTER VOICES: Each character has a distinct voice (see glossary).
   Stan is a fast-talking salesman ("vi snackar", "kompis").
   Guybrush is sarcastic and self-deprecating.
   Pirates are rough and colorful.
   Check Rule 6 in TRANSLATION_GUIDE.md for full voice descriptions.

6. NO INVENTED WORDS: Don't create compound words that don't exist in Swedish.
   "kompansen", "landlansen", "kanonkulleskansen" are NOT Swedish words.

GLOSSARY AND CONSISTENCY (Rules 7-10):
7. NAMES: Use Swedish versions per glossary (Rötbett, Järnkrok, Smilfink).
   Every occurrence — dialog, (13) name displays, OBNA entries, signed notes.

8. PRONOUNS: Always jag/mig/dig. Only use ja/mej/dej when the English
   character speaks in notably bad grammar.

9. GLOSSARY TERMS: Follow strictly:
   - Currency: "dubloner" (never "åttor")
   - Grog: "grogg" (double g)
   - Pulley: "trissa" (never "talja")
   - Idol: "idol" (never "avgud")
   - Landlubber: "landkrabba" (never "landlansen")
   - Compound words: no spaces ("voodookärlekspärlor" not "voodoo kärlekspärlor")

10. AMBIGUOUS WORDS: Before translating words like "crack", "handle", "chest",
    search the file for other occurrences to understand the actual meaning.

HUMOR (Rule 11):
11. PUNS: If the English contains a pun, create a Swedish pun. Literal
    translation of puns never works. Check pun_inventory.md for pre-planned
    solutions.

LENGTH (Rule 12):
12. No line text exceeds 256 characters. FA dialog choices must fit ~57 chars.

WORKFLOW — follow this exactly for your room:
1. Read ALL English strings for the room to understand the full context
2. Translate each string one at a time, giving full attention to each line
3. For each line, mentally check: false friends? calque? gender? glossary?
4. Check that OBNA names match their VERB references
5. Re-read all your Swedish strings in sequence — they should flow naturally
   as a player would experience them
6. Do a final pass checking every line against the critical rules above

Write the translated lines to swedish.txt, replacing the [E]-prefixed lines
for room [ROOM_NUMBER].
```

### Prompt: Room Reviewer (one per room, runs after translator)

```
You are a Swedish language reviewer checking a game translation for quality.
This is an authorized fan translation project.

You are reviewing ROOM [ROOM_NUMBER] ONLY.

Read these files carefully before starting your review:
- docs/TRANSLATION_GUIDE.md (ALL rules — you must understand them fully)
- translation/glossary.md (all vocabulary decisions and character voices)
- games/<GAME>/translation/swedish.txt (the translation to review)
- games/<GAME>/gen/strings/english.txt (English originals for comparison)

You did NOT write this translation. Review it with fresh eyes.

CHECK EVERY LINE in room [ROOM_NUMBER] against the full set of rules from
TRANSLATION_GUIDE.md. The most important checks, grouped by category:

FORMAT AND STRUCTURE:
□ Line prefix [ROOM:TYPE#RESNUM](OP) unchanged from English original
□ All control codes (\255\003, \255\001, \254\008, etc.) preserved exactly
□ Symbols (™, etc.) preserved exactly as in the English — not added or removed
□ @-only lines (buffer allocations) copied unchanged
□ Same number of \255\003 page breaks as the English
□ No line text exceeds 256 characters
□ All (FA) dialog choice lines fit in ~57 characters

NATURAL SWEDISH (Rules 1-3):
□ No lines that sound like translated English — check for calques
□ No calque patterns from the list in TRANSLATION_GUIDE.md Rule 3
  (e.g. "inte på humör för", "en fråga om stolthet", "var inte en främling")
□ No false friends: "hey" ≠ "hej" (this was the #1 error last time)
  "Hey" as exclamation → Oj/Åh/Hallå/Kolla/Hördu. "Hej" ONLY for greetings.
□ Generic messages (fallback strings for multiple objects) don't use den/det

GRAMMAR (Rules 4-5, 9):
□ Correct grammatical gender — den/det and adjective endings match en/ett noun
□ No invented compound words (every word must be real Swedish)
□ Always jag/mig/dig — no ja/mej/dej (unless English is ungrammatical)

CHARACTER AND TONE (Rule 6):
□ Character voice matches personality:
  - Stan: pushy salesman ("vi snackar", "kompis", rapid-fire)
  - Guybrush: sarcastic, self-deprecating, modern
  - Pirates: rough, colorful, use pirate slang
  - Check glossary for full voice descriptions

GLOSSARY COMPLIANCE (Rules 7, 10):
□ Character names use Swedish glossary form everywhere
  (Rötbett not Toothrot, Järnkrok not Meathook, Smilfink not Smirk)
□ Glossary terms used correctly:
  - "dubloner" not "åttor" for currency
  - "grogg" not "grog" (double g)
  - "trissa" not "talja" for pulley
  - "idol" not "avgud"
  - "landkrabba" not "landlansen"
□ Compound words have no spaces ("voodookärlekspärlor")
□ OBNA names match their VERB references for the same object in this room
□ Ambiguous words translated correctly in context (not in isolation)

HUMOR (Rule 11):
□ Puns create Swedish wordplay — not literal translations of English puns
□ Jokes land in Swedish — if the original is funny, the translation should be

FLOW:
□ Read all lines in room sequence — dialog flows naturally
□ No meaning changed from the English (only fix Swedish quality issues)

For each issue found:
1. Quote the line reference and current text
2. State which rule it violates
3. Provide the corrected text

Apply all fixes to swedish.txt.

If the room passes all checks with no issues, say so — don't invent problems.
```

---

## Phase 3 — Consistency Review

**Goal:** Verify every name, term, and object reference is consistent across the entire file.
**Input:** Complete `swedish.txt`, `glossary.md`
**Output:** Corrections applied in-place; `glossary.md` updated

### Prompt: Consistency Review

```
You are reviewing a complete Swedish game translation for cross-room consistency.
This is an authorized fan translation project.

Read:
- translation/glossary.md
- games/<GAME>/translation/swedish.txt

Perform these checks — use sub-agents to parallelize:

AGENT 1 — Name consistency:
- Grep for every character name variant (Toothrot, Tandröta, Rötbett, etc.)
  and verify ALL occurrences use the glossary form
- Check all (13) name display lines match the glossary
- Check initials in signed notes match Swedish names (H.T. → H.R. for Rötbett)
- Check "Kapten" vs "Kapen", "Svärdsmästaren" vs "Svärdmästaren"

AGENT 2 — Term consistency:
- Grep for: åttor, åtta (should be dubloner/dublon)
- Grep for: " grog[^g]" without double g (should be grogg)
- Grep for: ® (should be ™)
- Grep for: talja (should be trissa)
- Grep for: avgud (should be idol)
- Grep for: "[Hh]ej[!,.]" (likely should be oj/åh/hallå — check context)
- Grep for: " ja " at start of speech (should be "jag" unless intentional)
- Grep for: " mej " and " dej " (should be "mig" and "dig")

AGENT 3 — Object name consistency:
- For each OBNA line, find the corresponding VERB lines in the same room
- Verify the object noun matches between OBNA and VERB
- Check that the same object uses the same Swedish word across different rooms
- Flag mismatches

AGENT 4 — Insult system integrity:
- Verify all insult/comeback pairs still match after any edits
- Verify Sword Master insults map to correct comebacks
- Check that training insults in Smirk/Smilfink's dialog match the actual
  insult/comeback strings

Report all inconsistencies with line numbers. Fix them in swedish.txt.
```

---

## Phase 4 — Pun and Wordplay Polish

**Goal:** Review all flagged puns in context and improve awkward ones.
**Input:** `pun_inventory.md`, current `swedish.txt`
**Output:** Corrections applied in-place; `pun_inventory.md` annotated

### Prompt: Pun Polish

```
You are polishing the wordplay in a Swedish game translation. This is an
authorized fan translation project.

Read:
- games/<GAME>/translation/pun_inventory.md
- games/<GAME>/translation/swedish.txt
- translation/glossary.md

For each flagged pun in pun_inventory.md:
1. Read the Swedish translation in context (read adjacent strings in the
   same room/script — 5 lines before and after)
2. Does the joke land in Swedish? Is the comedy beat preserved?
3. If not, create a better Swedish pun. The Swedish joke can be completely
   different from the English as long as it's funny in context.

Key pun patterns from MI1 that needed fixing:
- piracy/conspiracy → pirateri/kons-PIRAT-ion (wordplay on shared syllable)
- "Booty for my beauty" → "Skatt för min skatt" (Swedish homophone)
- Book titles with body-part double meanings (head, leg, arm)
- "I'm double-parked" → "dubbelparkerad" (works directly in Swedish)

Update pun_inventory.md with resolution status for each entry.
```

---

## Phase 5 — Natural Language Review (4 story-arc agents)

**Goal:** Catch unnatural phrasing, calques, and stilted Swedish that sounds
like translated English. This was the largest category of fixes in MI1 (~150 fixes).
**Input:** Complete `swedish.txt`
**Output:** Corrections applied in-place

### Architecture

4 fresh agents, one per story arc. Each agent reads all rooms in their arc,
which gives them enough context to catch cross-room voice inconsistencies
(e.g. Stan sounding different in room 059 vs room 083) without being so much
text that quality drops.

This is NOT a correctness review — it's a naturalness review. Read as a native
Swedish speaker would.

### Prompt: Natural Language Reviewer (one per story arc)

```
You are a native Swedish language reviewer checking a game translation for
naturalness. This is an authorized fan translation project.

You are reviewing [STORY_ARC_DESCRIPTION] — rooms [ROOM_LIST].

Read:
- docs/TRANSLATION_GUIDE.md (especially Rules 1, 3, 5, 6)
- translation/glossary.md (character voices section)
- games/<GAME>/translation/swedish.txt (your assigned rooms)
- games/<GAME>/gen/strings/english.txt (English originals for comparison)

Review every line in your assigned rooms for NATURALNESS. For each line, ask:
"Would a native Swedish speaker say this, or does it sound like someone
thinking in English and writing in Swedish?"

Also check for VOICE CONSISTENCY: does each character sound the same across
all rooms in your arc? Stan should have the same energy in every scene.
Guybrush's sarcasm level shouldn't vary randomly between rooms.

Flag these specific patterns:

1. CALQUES: Swedish phrases that copy English structure
   Example: "Jag har en känsla av att" → "Jag tror" / "Jag har på känn att"
   Example: "Som en gest för att återställa" → "I ett försök att lappa ihop"

2. REGISTER MISMATCH: Character speaking in wrong tone
   Example: Stan saying "Titta bara" instead of "Kolla in"
   Example: Pirate saying formal Swedish instead of rough/colorful

3. STIFF PHRASING: Grammatically correct but unnatural
   Example: "Kanske du vill betala" → "Du kanske vill betala"
   Example: "Jag tror jag skulle få bättre resultat" → "Jag skulle nog få bättre resultat"

4. FALSE FRIEND SURVIVORS: "Hej" used as exclamation, etc.

5. LITERAL IDIOMS: English idioms translated word-for-word
   Example: "Var inte en främling" → "Titta in nån gång"
   Example: "Hej, kasta inte sten" → "Hördu, klaga inte på det"

6. FLAT TRANSLATIONS: Lines that are technically correct but lack the punch,
   humor, or personality of the original
   Example: "Ja, det är det näst största aphuvudet jag sett" →
            "Jösses, det är det näst största aphuvudet jag någonsin sett"

7. CROSS-ROOM VOICE DRIFT: Same character sounding different in different rooms
   Example: Stan using "vi pratar" in one room and "vi snackar" in another

IMPORTANT: Do NOT change the meaning of any line. Only fix Swedish quality
issues. If the English says "I'm lost" and the Swedish says "Jag är körd",
that's a good creative translation — don't change it back to "Jag har gått vilse".

For each issue, provide:
- Line reference [ROOM:TYPE#RESNUM]
- Current Swedish text
- Proposed fix
- Category (calque/register/stiff/false-friend/literal-idiom/flat/voice-drift)

Apply fixes to swedish.txt. If a room reads naturally with no issues, say so.
```

---

## Phase 6 — Length Validation

**Goal:** Ensure no string exceeds technical limits.
**Input:** `swedish.txt`
**Output:** Shortened strings applied in-place

### Prompt: Length Validation

```
Read games/<GAME>/translation/swedish.txt.

Check every line:
1. Extract the text portion after the ](XX) prefix
2. Count characters (including \255\003 as 2 chars each, other \NNN as 1 char)
3. Flag any line exceeding 256 characters total
4. Flag any (FA) line exceeding 57 visible characters

For flagged lines, shorten by:
1. Rephrasing (preferred — preserve meaning and tone)
2. Breaking with \255\003 (only if original uses similar pacing)
3. Last resort: cutting content (document what was removed)

Also verify: lines that are entirely @ characters are preserved unchanged.
```

---

## Phase 7 — Final Read-Through

**Goal:** Read the whole translation as a playthrough. Catch anything that
reads awkwardly in sequence even if each individual line looked fine in isolation.
**Input:** Complete, validated `swedish.txt`
**Output:** Final corrections applied in-place

### Prompt: Final Read-Through (one per game part)

```
You are doing the final read-through of a Swedish game translation. This is an
authorized fan translation project.

You are reading [GAME_PART] only.

Read games/<GAME>/translation/swedish.txt — focus on the rooms in your
assigned game part.

Read ALL strings for each room together as a sequence — this simulates how
a player experiences them. Check:

1. Dialog flows naturally from line to line
2. Characters stay in voice throughout a conversation
3. References to earlier events use consistent wording
4. No jarring tone shifts within a scene
5. Jokes land in context (a line might be fine alone but awkward in sequence)

This is the last pass. Only fix genuine problems — don't polish for the sake
of polishing. If a line is good enough, leave it.

Apply fixes to swedish.txt.
```

---

## Master Orchestrator Prompt (Phases 2-7)

Give this prompt to a single orchestrator agent after Phases 0 and 1 are
approved. It handles the entire pipeline automatically.

```
You are orchestrating the translation pipeline for a SCUMM adventure game.
This is an authorized fan translation project — the project owner has
purchased the game and has the legal right to create this Swedish translation.

Read docs/TRANSLATION_PLAN.md for the full workflow. You will execute
Phases 2 through 7 in order, spawning sub-agents for each.

The translation file is games/<GAME>/translation/swedish.txt. Lines prefixed
with [E] are untranslated. The glossary is at translation/glossary.md.

PHASE 2 — MAIN TRANSLATION:
Get the list of rooms from the [E]-prefixed lines in swedish.txt.
For EACH room, spawn a pair of agents in sequence:
  1. Translator agent — use the "Room Translator" prompt from the plan,
     filling in the room number. This agent translates that one room.
  2. Reviewer agent — use the "Room Reviewer" prompt from the plan.
     This agent reviews and fixes the translator's output for that room.
Run rooms in parallel (many translator→reviewer pairs at once).
Wait for ALL rooms to complete before moving to Phase 3.

PHASE 3 — CONSISTENCY REVIEW:
Spawn 4 sub-agents as described in the Phase 3 prompt (name consistency,
term consistency, object name consistency, insult system integrity).
Run all 4 in parallel. Wait for all to complete.

PHASE 4 — PUN POLISH:
Spawn 1 sub-agent with the Phase 4 prompt.
Wait for completion.

PHASE 5 — NATURAL LANGUAGE REVIEW:
Spawn 4 sub-agents, one per story arc:
  - Arc 1: rooms covering arrival on Mêlée and early exploration
  - Arc 2: rooms covering the three trials (swordfighting, treasure, thievery)
  - Arc 3: rooms covering getting a crew/ship and sailing to Monkey Island
  - Arc 4: rooms covering Monkey Island, return, and finale
Each agent uses the Phase 5 "Natural Language Reviewer" prompt with their
room list. Run all 4 in parallel.
Wait for all to complete.

PHASE 6 — LENGTH VALIDATION:
Spawn 1 sub-agent with the Phase 6 prompt.
Wait for completion.

PHASE 7 — FINAL READ-THROUGH:
Spawn 4 sub-agents, one per game part:
  - Part 1: rooms covering arrival and the three trials
  - Part 2: rooms covering getting a ship and crew
  - Part 3: rooms covering Monkey Island
  - Part 4: rooms covering return and finale
Run all 4 in parallel. Wait for all to complete.

After Phase 7, run the build and tests:
  cd games/<GAME> && bash ../../scripts/build.sh
  cd games/<GAME> && bash ../../scripts/test.sh --all

Report the final status.

IMPORTANT: Each sub-agent must be a NEW agent — do not reuse agents across
phases. Fresh agents with no memory of prior work catch more errors.
```

---

## Phase Summary

| Phase | Goal | Agent model | Depends on |
|-------|------|-------------|------------|
| 0 | Glossary + pun inventory | Parallel scan agents | — |
| 1 | Insult swordfighting | 1 creative agent | Phase 0 + user approval |
| 2 | Main translation | 1 translator + 1 reviewer per room (parallel rooms) | Phase 1 + user approval |
| 3 | Consistency review | 4 parallel check agents | Phase 2 |
| 4 | Pun polish | 1 agent | Phase 3 |
| 5 | Natural language review | 4 agents (one per story arc) | Phase 3 |
| 6 | Length validation | 1 agent | Phases 4, 5 |
| 7 | Final read-through | 4 agents (one per game part) | Phase 6 |

**Total agents for a ~90-room game:**
- Phase 2: ~180 agents (90 translators + 90 reviewers)
- Phase 3: 4 agents
- Phase 4: 1 agent
- Phase 5: 4 agents
- Phase 6: 1 agent
- Phase 7: 4 agents
- **Total: ~194 agent spawns**

---

## Error Prevention Summary

These are the most common errors from the MI1 translation, ranked by frequency.
The paired translator/reviewer model in Phase 2 is designed to catch most of
these at the source, with Phases 3-5 as safety nets:

| # | Error type | Count in MI1 | Caught by |
|---|-----------|-------------|-----------|
| 1 | Stiff/formal phrasing, calques | 90+ | Phase 2 reviewer, Phase 5 |
| 2 | Symbols not preserved from original | 60+ | Phase 2 translator (preserve), Phase 3 |
| 3 | "Hey" → "Hej" (false friend) | 30+ | Phase 2 reviewer, Phase 5 |
| 4 | ja/mej/dej instead of jag/mig/dig | 40+ | Phase 2 reviewer, Phase 3 |
| 5 | Character names not translated | 30+ | Phase 2 reviewer, Phase 3 |
| 6 | Wrong glossary terms (åttor, grog, talja, avgud) | 35+ | Phase 2 reviewer, Phase 3 |
| 7 | FA lines too long | 20+ | Phase 2 reviewer, Phase 6 |
| 8 | Gender errors (den/det) | 15+ | Phase 2 reviewer |
| 9 | Object name inconsistency (OBNA vs VERB) | 15+ | Phase 2 reviewer, Phase 3 |
| 10 | Invented compound words | 10+ | Phase 2 reviewer |
| 11 | Puns translated literally | 10+ | Phase 4 |
| 12 | Character voice drift across rooms | ~20 | Phase 5 |
| 13 | Meaning changed during review | ~10 | Phase 5, Phase 7 |

---

## File Layout

```
games/<game>/translation/
  swedish.txt          — The translation file (scummtr format, UTF-8)
  pun_inventory.md     — Language-dependent strings: English original + Swedish solution

translation/
  glossary.md          — Proper nouns, recurring terms, translation decisions (shared)
```

The `swedish.txt` file is the only file that feeds into the build pipeline.

---

## Build and Test

After completing all phases:

```bash
cd games/<game>
bash ../../scripts/build.sh        # Builds patcher + applies @ padding
bash ../../scripts/test.sh --all   # Runs all tests including integration
```

The build automatically pads object names with `@` for SE buffer safety.
The source `swedish.txt` is never modified by the build.

---

## Dynamic Name Padding

Some object names change at runtime (e.g. "mug" → "mug of grog"). The SE
engine writes replacement names in-place with no bounds check.

**Translators don't need to do anything about this.** The build pipeline
handles it automatically via `tools/calc_padding.py`.

---

## Notes on Specific Challenges

### "Monkey Island" — the island name
Keep as "Monkey Island" (the game's own title is the established name).
Use ™ (`\153`) after it in all in-game references.

### The SCUMM Bar
Keep "SCUMM Bar" — it's a meta-joke referencing the engine.

### Grog
"Grog" → "grogg" in Swedish (double g). Translate descriptions around it.

### Text speed
Swedish text is longer than English. Set ScummVM text speed to 120+ or use
the SE's auto-advance feature. The SE's timed text display may cut off longer
strings — test in-game.
