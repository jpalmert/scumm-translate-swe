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

### Use natural Swedish, not literal translations

Read the translation aloud — if it sounds like translated English, rewrite it.
Common mistakes:

- Prefer compound words: ❌ "rustigt igenstängda" → ✅ "igenrostade"
- Prefer common verbs: ❌ "svimmade av" → ✅ "tuppade av"; ❌ "fatta eld" → ✅ "brinna"
- Adverb placement: Swedish often puts adverbs after the object — ❌ "Jag bränner bara allt" → ✅ "Jag bränner allt bara"
- Single spaces after periods (not double)

### Generic messages — avoid gender-specific pronouns

Messages that apply to many different objects (room 010 fallback strings, etc.) must not
assume a grammatical gender. Rewrite to omit the pronoun entirely:

- ❌ "Jag kan inte nå **det**." → ✅ "Jag kan inte nå."
- ❌ "Jag ser inget speciellt med **det**." → ✅ "Jag ser inget speciellt."
- ❌ "Jag kan inte flytta **det**." → ✅ "Jag kan inte flytta."

The object is implied from context. Guessing "den" or "det" will be wrong half the time.

### Research ambiguous words before translating

Before translating an ambiguous word (e.g. "crack", "handle", "chest"), grep the full
translation file for other occurrences — VERB entries, dialog lines, other rooms — to
determine the actual object and meaning. Don't translate in isolation.

### Specific phrase: the game title

"the Secret of Monkey Island®" → **"Monkey Islands® Hemlighet"** (capital S)
"the secret of Monkey Island®" → **"Monkey Islands® hemlighet"** (lowercase)

Swedish word order: possessor first — "Monkey Islands® Hemlighet" not "Hemligheten på Monkey Island®".
