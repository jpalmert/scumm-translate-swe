# Bug Tracker

## Open Bugs

### BUG-001: SE — `^` appears literally in dialog (e.g. "Öh^")
**Status:** LIKELY MISDIAGNOSED — under review  
**Affects:** Special Edition (SE) graphics mode  
**Symptom:** A visible `^` caret appeared in SE dialog. Initially attributed to the SE engine not handling the SCUMM newline control byte.  
**Updated analysis:** `^` appears in the English extracted text too, which means the SE engine DOES handle the SCUMM newline control byte correctly. 
The visible `^` was likely NOT a newline handling bug. More probable cause: the Ö glyph (SCUMM code 93) was rendering as `^` or similar due to a font 
lookup issue — codes 93 (Ö) and 94 (`^`) are adjacent in the ASCII table and if the font lookup was off-by-one or writing to the wrong address, Ö 
would render as `^`. This would be a manifestation of the SE font patching being applied to the wrong game version or having a bug in the lookup offset.  
**Retest needed:** After confirming BUG-002 charset fix and SE font patching correctness, verify whether this bug still occurs.

---

### BUG-002: Classic — Swedish characters display as wrong glyphs in dialog
**Status:** FIXED — charset patch implemented  
**Affects:** Classic SCUMM mode (ScummVM / SE F10 mode)  
**Symptom:** Wrong characters appear in dialog (å, ö, Ä show wrong glyphs or nothing).  
**Root cause:** CHAR_0001 (verb/menu charset) and CHAR_0003 (small text) in `MONKEY1.001` lacked Swedish glyph bitmaps at SCUMM code positions 91(Å), 92(Ä), 93(Ö), 123(å), 124(ä), 125(ö). CHAR_0002 (dialog) already had them in the GOG/Steam SE release.  
**Fix:** `internal/charset` package embeds pre-computed patched CHAR_0001 (+28 bytes) and CHAR_0003 (+78 bytes) binaries. `PatchMonkey1001()` splices them in and updates LECF/LFLF container sizes. `PatchMonkey1000()` updates the DCHR offset table so MONKEY1.000 points to the correct shifted positions. Both patches are applied in `se.go` (Steps 5a/5b) and `classic.go`.

---

### BUG-003: Classic — Menu "Öppna" loses leading Ö (shows "ppna")
**Status:** ROOT CAUSE IDENTIFIED — two contributing factors  
**Affects:** Classic SCUMM mode (ScummVM), verb/menu rendering  
**Symptom:** "Öppna" (Swedish for "Open") shows as "ppna" in the in-game menu.  
**Root cause:**  
1. **`-A aov` flag** (FIXED — see BUG-R04): Our scummtr invocation previously used `-A aov` which prevents verb/object/actor string injection entirely. Without this fix, "Öppna" would never be injected at all.  
2. **Charset not patched** (see BUG-002): Even with verb injection, the Ö glyph (code 93) cannot render until the charset is patched. The Ö is present in the string but has no bitmap → renders as nothing → "ppna".  
**Fix:** BUG-R04 already removes `-A aov`. Full fix requires charset patching (BUG-002).

---

### BUG-004: Classic — Swedish chars show as empty rectangles in verb selection bar
**Status:** ROOT CAUSE IDENTIFIED — same as BUG-002/003  
**Affects:** Classic SCUMM mode (ScummVM), verb selection / action text at top of screen  
**Symptom:** When selecting an action, Swedish characters display as empty rectangles.  
**Root cause:** Same as BUG-002 and BUG-003 — no charset patch means no bitmaps for codes 91-93/123-125/130, which renders as empty glyphs. Verb injection was also blocked by `-A aov` (fixed in BUG-R04).  
**Fix:** Charset patching (BUG-002).

---

## Resolved Bugs

### BUG-R01: Windows — "not a valid Win32 application" error
**Status:** RESOLVED  
**Was:** `scummtr-windows-x64.exe` asset was an empty placeholder file. Fix: replaced with real PE32 binary from scummtr v0.5.1 release.

### BUG-R02: SE — Swedish characters showing as wrong glyphs
**Status:** RESOLVED  
**Was:** scummtr `-c` flag did not reliably map Windows-1252 Swedish chars to the correct SCUMM codes for `monkeycdalt`. Fix: pre-encode UTF-8 Swedish characters to SCUMM escape codes (`\091`–`\130`) before passing to scummtr, matching the monkeycd_swe approach.

### BUG-R03: Translation file displayed as squares in editor
**Status:** RESOLVED  
**Was:** `monkey1.txt` was Windows-1252 encoded. Fix: converted to UTF-8 via `iconv`; updated encoder to use `strings.ReplaceAll` on UTF-8 input.

### BUG-R04: Classic — verb/object/actor strings not injected (menu not translated)
**Status:** RESOLVED  
**Was:** `classic.go` used `-A aov` flag with scummtr, which prevents injection of verb, object, and actor name strings. This meant menu items like "Öppna" were never written. Fix: removed `-A aov` from the scummtr invocation. (thanius/monkeycd_swe does not use this flag either.)
