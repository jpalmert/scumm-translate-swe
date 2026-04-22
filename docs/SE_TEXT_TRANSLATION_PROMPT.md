# SE Supplemental Text Translation — Orchestrator Prompt

This is a single prompt for translating the SE-specific hint text and UI text
files. Give this to one orchestrator agent. It will spawn sub-agents for each
phase automatically.

Unlike the main game translation (see `TRANSLATION_PLAN.md`), these files are
small enough that the whole pipeline can run in one session with no human
checkpoints needed.

---

## The Prompt

```
You are orchestrating the translation of two supplemental text files for the
Monkey Island 1 Special Edition Swedish translation. This is an authorized fan
translation project — the project owner has purchased the game and has the
legal right to create this translation.

The MAIN game translation (games/monkey1/translation/swedish.txt) is already
complete. These two files contain SE-specific text that must be translated to
match the main translation's vocabulary and style.

FILES TO TRANSLATE:
- games/monkey1/translation/hints_swedish.txt (517 hint strings, ADDR<TAB>text)
- games/monkey1/translation/uitext_swedish.txt (125 UI strings, KEY<TAB>text)

Lines prefixed with [E] are untranslated. Remove the [E] prefix and replace
the English text with Swedish.

BEFORE ANY TRANSLATION, read these files carefully:
- translation/glossary.md (all vocabulary decisions — follow strictly)
- docs/TRANSLATION_GUIDE.md (Swedish language rules 1-12)
- games/monkey1/translation/swedish.txt (the complete main translation —
  you MUST use the same words for characters, items, locations, and concepts)

==========================================================================
PHASE 1 — VOCABULARY EXTRACTION
==========================================================================

Spawn a sub-agent to build a lookup table of how key terms were translated
in the main swedish.txt. This is critical because the hints reference the
same characters, items, and locations as the game dialog.

The agent should grep swedish.txt and build a mapping including at least:

  Characters:
  - Sword Master → Svärdsmästaren
  - Herman Toothrot → Herman Rötbett
  - Meathook → Järnkrok
  - Captain Smirk → Kapten Smilfink
  - Fettucini Brothers → Fettucinibröderna
  - Governor → Guvernören
  - Stan → Stan
  - Otis → Otis
  - Carla → Carla
  - Voodoo Lady → (check swedish.txt for exact form)

  Items:
  - rubber chicken with a pulley in the middle → gummikyckling med en trissa i mitten
  - breath mints → mintpastiller
  - root beer → rotöl
  - grog → grogg
  - idol / fabulous idol → idol / fabulös idol
  - piranha poodles → pirayapudlar
  - shovel → spade
  - sword → svärd
  - compass → kompass
  - pieces of eight → dubloner
  - banana picker → bananplockare
  - head of the navigator → (check swedish.txt for exact form)
  - necklace → halsband
  - Gopher Repellent → (check swedish.txt for exact form)

  Locations:
  - SCUMM Bar → SCUMM Bar
  - Governor's Mansion → Guvernörens herrgård
  - Stan's Previously Owned Vessels → Stans Begagnade Fartyg
  - Monkey Island → Monkey Island
  - Mêlée Island → Mêlée Island
  - Giant Monkey Head → (check swedish.txt for exact form)

  Game concepts:
  - Sword Master's house → (check swedish.txt)
  - the three trials → de tre prövningarna
  - crew → besättning
  - insults/retorts → förolämpningar/repliker

Save this mapping — it will be passed to the translator agents.

==========================================================================
PHASE 2 — TRANSLATE HINT TEXT
==========================================================================

Spawn sub-agents to translate hints_swedish.txt. Split into 3 agents by
content area for manageable batch sizes:

AGENT A: Lines 49744-80000 (Part 1-2 hints: Mêlée Island, trials)
AGENT B: Lines 80001-150000 (Part 2-3 hints: ship, Monkey Island puzzles)
AGENT C: Lines 150001-220000 (Part 3-4 hints: more MI puzzles, return, alternate paths)

Each translator agent gets the vocabulary mapping from Phase 1 and these
instructions:

  HINT TRANSLATION RULES:

  1. USE THE EXACT SAME WORDS as the main translation (swedish.txt).
     If swedish.txt calls it "Svärdsmästaren", the hint must too.
     If swedish.txt calls it "pirayapudlar", the hint must too.
     NEVER invent your own translation for a term that already exists.

  2. Hints are instructional — clear and direct. They don't need the
     character voice or humor of the main dialog. But they should still
     be natural Swedish, not translated English.

  3. Apply ALL Swedish language rules from TRANSLATION_GUIDE.md:
     - No "hej" as exclamation
     - No calques or English sentence structure
     - Correct grammatical gender
     - jag/mig/dig (not ja/mej/dej)
     - "grogg" not "grog", "dubloner" not "åttor", etc.

  4. Preserve the ADDR<TAB> prefix exactly. Only translate the text after
     the tab character. Remove the [E] prefix.

  5. Some hints reference game mechanics ("Click Open on the Door").
     Translate the verb names to match the in-game verb buttons from
     swedish.txt (check room 010 SCRP#0022 entries for verb translations).

After each agent finishes, spawn a REVIEWER agent for their section.
The reviewer checks:
  □ Every character/item/location name matches the vocabulary mapping
  □ Natural Swedish (no calques, no false friends)
  □ Correct gender (den/det)
  □ Verb names match the in-game verbs from swedish.txt
  □ No [E] prefixes remaining
  □ ADDR<TAB> format preserved

==========================================================================
PHASE 3 — TRANSLATE UI TEXT
==========================================================================

Spawn ONE agent to translate uitext_swedish.txt. This is only 125 lines.

  UI TEXT TRANSLATION RULES:

  1. MENU LABELS: Translate to standard Swedish game UI terminology.
     - Save Game → Spara spel
     - Load Game → Ladda spel
     - Settings → Inställningar
     - Yes/No → Ja/Nej
     - etc.

  2. KEEP IN ENGLISH: Platform-specific terms that Swedish gamers expect
     in English:
     - "Xbox 360", "Xbox LIVE", "Xbox Guide button"
     - "Leaderboard" (widely used in Swedish gaming)
     - Technical terms like "Storage device" can stay or translate
     - "Gamertag" stays English

  3. GAME PART TITLES: Must sound good as chapter titles.
     - "Part One: The Three Trials" → use the established Swedish
       translation of "the three trials" from the glossary
     - "The Last Part: Guybrush Kicks Butt" → translate creatively

  4. OVERLAY VERBS: Must match the in-game verb buttons exactly.
     Check swedish.txt room 010 SCRP#0022 entries for:
     Open, Use, Pick Up, Push, Pull, Close, Look At, Talk To, Give

  5. CREDITS TEXT: Translate section titles (these appear as story titles).
     - "Deep In The Caribbean" → translate
     - "The Island of Mêlée" → translate (keep "Mêlée" as-is)

  6. Preserve KEY<TAB> format exactly. Only translate after the tab.
     Remove [E] prefix.

  7. WARNING/ERROR messages: Translate clearly. Players need to understand
     these. Keep them concise.

After translation, spawn a REVIEWER agent that checks:
  □ Overlay verbs match in-game verb buttons from swedish.txt
  □ Platform terms handled correctly (kept English where appropriate)
  □ Warning messages are clear and complete
  □ No [E] prefixes remaining
  □ KEY<TAB> format preserved

==========================================================================
PHASE 4 — CROSS-FILE CONSISTENCY CHECK
==========================================================================

Spawn ONE agent to do a final consistency check across all three files:

  1. Grep hints_swedish.txt for every character name, item name, and
     location name. Verify each matches swedish.txt exactly.

  2. Grep uitext_swedish.txt overlay verbs and verify they match the
     verb buttons in swedish.txt.

  3. Check that "grogg", "dubloner", "trissa", "idol", and other glossary
     terms are used correctly in both files.

  4. Check that no [E] prefixes remain in either file.

Fix any inconsistencies found.

==========================================================================
DONE
==========================================================================

After Phase 4, report what was translated and any decisions that were made.
```
