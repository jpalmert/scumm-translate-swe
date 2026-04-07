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
| `\015` | ® | Trademark glyph — appears after game titles ("Monkey Island`\015`") |
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

## Fixed-width slots

Some strings are internal slot labels with no visible dialog content.
They may appear empty or contain only control codes — leave them as-is.

---

## Swedish character encoding

Swedish characters are stored using their Latin-1 code points. The injection
tooling handles encoding automatically; write normal UTF-8 Swedish in the
translation file.

Supported characters: **Å Ä Ö å ä ö é**
