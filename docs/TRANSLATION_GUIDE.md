# Translation Guide

This document describes the format of the strings in `english.txt` and the rules
for producing a correct Swedish translation file.

---

## File format

Each line is either blank or a string entry:

```
[ROOM:TYPE#RESNUM](OP)text content
```

- `[ROOM:TYPE#RESNUM]` — identifies where the string lives in the game; preserve unchanged
- `(OP)` — the SCUMM opcode that uses this string (see below); preserve unchanged
- `text content` — the part to translate

---

## Opcodes

The opcode in `(XX)` tells you what kind of string it is and who is speaking.

| Opcode | Type | Description |
|--------|------|-------------|
| `(D8)` | Dialog | **Guybrush speaking** — the most common dialog opcode |
| `(14)` | Dialog | **NPC speaking** — any non-player character |
| `(94)` | Dialog | **NPC speaking** — same as `(14)`, different internal argument encoding |
| `(FA)` | Dialog | **Guybrush speaking** — verb interaction responses ("I want it.", "How much is it?") |
| `(13)` | Name | **Actor name** — sets a character's display name dynamically |
| `(93)` | Name | **Actor name** — same as `(13)`, different internal argument encoding |
| `(54)` | Name | **Object name** — sets an object's display name dynamically ("a few memos", "a pile of memos") |
| `(D4)` | Name | **Object name** — same as `(54)`, different internal argument encoding |
| `(7A)` | UI | **Verb label** — the action verbs shown in the game UI ("Hypnotize", "Throw") |
| `(27)` | Mixed | **Internal/system strings** — some are real dialog (e.g. sword-fight insults), some are padding |
| `(__)` | Name | **Static object name** — baked into the resource, not set by a script |

The opcode variants `(94)`, `(D4)`, `(93)` and `(FA)` are functionally identical to
their counterparts `(14)`, `(54)`, `(13)` and `(7A)` — the difference is only in
how SCUMM internally encodes the argument, which is invisible to the translator.

---

## Control codes

The text contains embedded control codes. **Preserve them exactly** — do not
translate, reorder, or remove them. Only translate the human-readable text
around them.

### `\255` — primary control prefix (0xFF)

Always followed by a sub-code:

| Sequence | Meaning |
|----------|---------|
| `\255\001` | **Newline** — line break within the same speech bubble |
| `\255\002` | **Page/scene end** — end of a credits card or scene title; advances without a click |
| `\255\003` | **Clear text** — closes the speech bubble and waits for click (or auto-advances) before showing the next one |
| `\255\004\NNN\NNN` | **Print integer** — inserts a number from a script variable; the two bytes after are an index, leave them unchanged |
| `\255\007\NNN\NNN` | **Print string** — inserts a string from a script variable; leave unchanged |

`\255\005`, `\255\006` and `\255\008` appear only in internal debug/script strings
that do not need translation.

### `\254` — secondary control prefix (0xFE)

| Sequence | Meaning |
|----------|---------|
| `\254\001` | **Soft newline** — indented line break, used in signs and dictionary-style text |
| `\254\008` | **Soft wrap** — continuation indent for long lines |

### Special character bytes

| Code | Character | Notes |
|------|-----------|-------|
| `\153` | ™ | Trademark glyph — appears after game titles ("Monkey Island™") |
| `\250` | non-breaking space | Keeps names together in credits |
| `\136` | ê | Part of "Mêlée" — preserve as-is |
| `\130` | é | Accent character — preserve as-is |

---

## Ellipsis and trailing-off speech

The English text uses `...` for hesitation and trailing-off speech:

```
Hmmmm...\255\003I loved this stuff when I was a kid.
Er...\255\003That's not exactly what I meant.
```

Translate these naturally. You may use `...` wherever the character trails off,
hesitates, or the sentence continues on the next page.

---

## Multi-page speech

A single speech act is often split across several lines using `\255\003`:

```
[020:SCRP#0095]Well, I sailed here with a friend of mine twenty years ago.\255\003We hoped to discover the Secret of Monkey Island\015.\255\003But my friend met with a horrifying and tragic accident...\255\003...which claimed his life...\255\003...and I couldn't sail the ship back by myself.
```

Each `\255\003` is a page break — the player clicks through them. Keep the
same number of pages in the translation; don't merge or split them.

A line starting with `...` continues directly from the previous page:

```
...which claimed his life...
```

---

## The `@` character

`@` is invisible in the SCUMM engine (zero width, never rendered). Some
English lines contain `@` characters as padding.

**Lines that are entirely `@` with no visible text must be kept as-is.**
These are `PutCodeInString` buffer allocations — the game writes binary
data into them at specific offsets. Removing the `@` causes the buffer
to be too small, which corrupts memory (menu colors in ScummVM, save
crashes in the SE). Example:

```
[010:SCRP#0001](27)@@@@@@@@@@@@
[010:SCRP#0001](27)@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
```

**Trailing `@` on object/actor names probably doesn't need preserving.**
Testing showed no issues when removing it. Translate the visible text
normally and ignore the `@`.

---

## Swedish character encoding

Swedish characters are stored using their Latin-1 code points. The injection
tooling handles encoding automatically; write normal UTF-8 Swedish in the
translation file.

Supported characters: **Å Ä Ö å ä ö é**

---

## Swedish language rules

These rules are distilled from the complete MI1 translation, where hundreds of
fixes were needed after the initial pass. Each section addresses a specific
category of error that appeared repeatedly.

---

### Rule 1: Use natural Swedish, not literal translations

Read the translation aloud — if it sounds like translated English, rewrite it.
Common mistakes:

- Prefer compound words: ❌ "rustigt igenstängda" → ✅ "igenrostade"
- Prefer common verbs: ❌ "svimmade av" → ✅ "tuppade av"; ❌ "fatta eld" → ✅ "brinna"
- Adverb placement: Swedish often puts adverbs after the object — ❌ "Jag bränner bara allt" → ✅ "Jag bränner allt bara"
- Single spaces after periods (not double)
- Don't calque English sentence structure — Swedish has V2 word order (verb second in main clauses)

**Examples from MI1 fixes:**

| Initial (calque) | Fixed (natural Swedish) |
|-------------------|------------------------|
| Jag har en känsla av att... | Jag tror... / Jag har på känn att... |
| Det var nära, men jag tror inte det träffar igen. Det skottet var ett på miljonen! | Det var nära, men jag tror inte det träffar igen. Det var en träff på miljonen! |
| Som en gest för att återställa vår vänskap | I ett försök att lappa ihop vår vänskap |
| Du har mycket fräckheter som kommer till den här stan | Du har mage att komma till den här stan |
| Oddsen måste vara otroliga! | Chansen måste vara otroligt liten! |
| Jag skulle älska att få dig uppstoppad | Att stoppa upp dig hade gjort mig rik |
| Vid närmare eftertanke kanske det här inte är skeppet för mig | Det här är nog inte skeppet för mig ändå |

---

### Rule 2: Generic messages — avoid gender-specific pronouns

Messages that apply to many different objects (room 010 fallback strings, etc.) must not
assume a grammatical gender. Rewrite to omit the pronoun entirely:

- ❌ "Jag kan inte nå **det**." → ✅ "Jag kan inte nå."
- ❌ "Jag ser inget speciellt med **det**." → ✅ "Jag ser inget speciellt."
- ❌ "Jag kan inte flytta **det**." → ✅ "Jag kan inte flytta."

The object is implied from context. Guessing "den" or "det" will be wrong half the time.

---

### Rule 3: Beware of false friends and calques

Swedish and English have many words that look similar but mean different things.
Always translate based on the **Swedish meaning**, not visual similarity to the English.

**False friends** (words that look alike but differ):

| English | Looks like Swedish | Actually means in Swedish | Correct Swedish |
|---------|--------------------|--------------------------|-----------------|
| hey (exclamation) | hej | hello (greeting) | oj, åh, hallå, kolla |
| gift | gift | married / poison | gåva, present |
| chef | chef | boss/manager | kock |
| eventually | eventuellt | possibly/perhaps | till slut, så småningom |
| actual | aktuell | current/relevant | faktisk, verklig |

Note: "obekväm" works for both physical and social discomfort in Swedish — it is NOT a false friend.

**"Hey" vs "Hej"** was the single most frequent error in the MI1 translation
(30+ instances fixed). "Hey" in English is an exclamation to get attention or
express surprise — it is NOT a greeting. Translate based on context:
- Surprise/discovery: "Oj!", "Åh!", "Kolla!"
- Getting attention: "Hallå!", "Hördu!"
- Casual/dismissive: "Äh", "Tja", "Visst"
- Only use "Hej" when the English is actually "Hi"/"Hello" (a greeting)

**Common calque patterns** to avoid:

| English pattern | Calque (wrong) | Natural Swedish |
|-----------------|----------------|-----------------|
| not in the mood for | inte på humör för | inte upplagd för |
| a matter of pride | en fråga om stolthet | en hederssak |
| bad feeling | dålig känsla/aning | onda aningar |
| come to your senses | komma till sans (= regain consciousness) | besinna sig, ta sitt förnuft till fånga |
| it's your loss | det är din förlust | du som förlorar på det |
| until all hours | till alla tider | in på småtimmarna |
| escape artist | flyktartist | utbrytarkung/utbrytarkonstnär |
| don't be a stranger | var inte en främling | titta in nån gång |
| to carry (move sth) | att bära | att flytta |
| don't push it | driva det längre | chansa |
| that's sweet | det är söt (adj. for person) | det är sött (neuter: about the situation) |
| standard potion | standarddryck för exorcism | vanliga exorcismdryck |
| I'm lost | jag är förlorad | jag är körd / jag har gått vilse |

When in doubt, read the Swedish aloud — if it sounds like something a Swede
would only say because they're thinking in English, rewrite it.

---

### Rule 4: Grammatical gender consistency

Swedish nouns are either en-words (common) or ett-words (neuter). Getting this wrong
is immediately noticeable. The initial MI1 translation had dozens of gender errors.

**Common errors:**
- "den är svetsad" on an ett-word → "**det** är svetsat" / "**den** är svetsad" (match the noun)
- "Jag tror inte **den** rymmer" (about a stop/seat) → "Jag tror inte **det** rymmer" (stop is ett-word)
- "ett fabulös dörrstopp" → "ett fabulöst dörrstopp" (neuter adjective)

**Before translating:** check whether the object's Swedish noun is en or ett.
When referring to previously mentioned objects, use the correct pronoun.

---

### Rule 5: Don't invent words or compound expressions

The initial translation created several non-existent Swedish words:

| Invented | Problem | Correct |
|----------|---------|---------|
| kompansen | -ansen is not a Swedish suffix | kompis |
| kanonkulleskansen | non-word compound | kanonkulan |
| landlansen | non-word | landkrabba |
| piranjapadel | not the right pun | pirayapudel (piranha + poodle) |
| muterande besättning | "mutating crew" | myterisk besättning |
| syltingtallrik | not a real compound | sylta |
| fransyska (as food) | means "French woman" | stek |

**Rule:** If a Swedish compound word doesn't appear in SAOL or common usage,
don't use it. Find a real word instead.

---

### Rule 6: Character voice consistency

Each character has a distinct voice. The initial translation often flattened
characters to the same neutral register.

**Stan (used ship salesman)** — energetic, pushy, over-the-top:
- Uses "vi snackar" not "vi pratar" (punchier)
- Uses "kompis" not formal address
- "Kolla in" not "Titta på", "schysst" not "rimligt"
- Every sentence should feel like sales pressure

**Guybrush** — modern, sarcastic, slightly naive:
- Uses "Oj" or "Åh" for surprise (never "Hej" unless greeting someone)
- Self-deprecating humor: "Jag är körd" not "Jag är förlorad"
- Casual but grammatically correct: jag/mig/dig, not ja/mej/dej

**Pirates** — rough, colorful:
- Pirate greeting: "Ohoy", "Ahoj" (never "Aja" which sounds like resigned sighing)
- Curses: "förbaskat", "tusan", "dödskallar"

**Fester Shinetop** — threatening, menacing:
- Short declarative sentences
- "apungen" not "apgransen" for insults

---

### Rule 7: Translate character names per glossary

Character names with descriptive meanings must be translated. The initial MI1
translation left several in English or used wrong translations:

| English | Initial (wrong) | Correct |
|---------|-----------------|---------|
| Herman Toothrot | Herman Toothrot | **Herman Rötbett** |
| Meathook | Meathook | **Järnkrok** |
| Captain Smirk | Kapten Smirk | **Kapten Smilfink** |
| Herman Tandröta | (inconsistent variant) | **Herman Rötbett** |

**Every occurrence** of the name must use the Swedish version — in dialog, in
`(13)` name displays, in `OBNA` entries, in signed notes (initials too: H.T. → H.R.).

---

### Rule 8: Trademark symbols

Use ™ (`\153` in SCUMM) for all in-game brand references. Never ®.
See `glossary.md` → "Brand References / Trademark Symbols" for the full list.

---

### Rule 9: Pronoun formality (ja/mej/dej vs jag/mig/dig)

Sloppy pronoun forms (ja, mej, dej) should **only** be used when the English
character is speaking in notably bad grammar. Standard dialog — even for pirates
— uses jag/mig/dig.

The initial MI1 translation used ja/mej/dej throughout room 083 (Stan's dock
scene). Every instance was corrected to jag/mig/dig because Stan speaks fast
but grammatically.

---

### Rule 10: Research ambiguous words before translating

Before translating an ambiguous word (e.g. "crack", "handle", "chest"), grep the full
translation file for other occurrences — VERB entries, dialog lines, other rooms — to
determine the actual object and meaning. Don't translate in isolation.

**Example from MI1:** "stool" appeared as bar furniture — initial translation used
"mugg" (mug/cup) in some places instead of "stop" (bar stool).

---

### Rule 11: Preserve puns and jokes, don't translate literally

When the English contains a pun, the Swedish must also contain a pun — even if
it's a completely different joke. Literal translation of puns almost never works.

**Examples from MI1 fixes:**

| English (pun) | Initial (literal, broken) | Fixed (Swedish pun) |
|---------------|---------------------------|---------------------|
| piracy / conspiracy | sjöröveri / KONSPIR-ation | pirateri / kons-PIRAT-ion |
| "Booty for my beauty" | "Byte för mitt snygge" | "Skatt för min skatt" |
| "I'm double-parked" | "dubbelförbjuden" | "dubbelparkerad" (works in Swedish too) |
| "How you get ahead in navigation" | "Hur du kommer framåt" | "Hur du får ett huvud för navigering" (huvud=head) |
| "How to get a leg up in treasure hunting" | "Hur du får ett ben upp" | "Hur du får fotfäste i skattjakten" |

---

### Rule 12: FA dialog lines have a character limit

Lines with opcode `(FA)` are player dialog choices shown in a selection box.
The box is approximately **57 characters wide**. Lines exceeding this get
truncated in-game. Always check FA line length and shorten if needed.

---

### Specific phrase: the game title

"the Secret of Monkey Island™" → **"Monkey Islands™ Hemlighet"** (capital H)

Swedish word order: possessor first — "Monkey Islands™ Hemlighet" not "Hemligheten på Monkey Island™".
