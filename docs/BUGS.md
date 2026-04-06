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
**Retest needed:** After confirming SE font patching correctness, verify whether this bug still occurs.

---

## Resolved Bugs

### BUG-002: Classic — Swedish characters display as wrong glyphs in dialog
**Status:** RESOLVED  
**Was:** CHAR_0001 (verb/menu charset) and CHAR_0003 (small text) in `MONKEY1.001` lacked Swedish glyph bitmaps at SCUMM code positions 91(Å), 92(Ä), 93(Ö), 123(å), 124(ä), 125(ö).  
**Fix:** `internal/charset` package embeds pre-computed patched CHAR block binaries. Applied via scummrp at runtime.

### BUG-003: Classic — Menu "Öppna" loses leading Ö (shows "ppna")
**Status:** RESOLVED  
**Was:** Two causes: (1) scummtr invoked with `-A aov` which blocks verb/object/actor string injection entirely; (2) Ö glyph (SCUMM code 93) had no bitmap in the charset → rendered as nothing.  
**Fix:** Removed `-A aov` (BUG-R04); charset patched (BUG-002).

### BUG-004: Classic — Swedish chars show as empty rectangles in verb selection bar
**Status:** RESOLVED  
**Was:** Same root cause as BUG-002/003 — missing charset bitmaps and blocked verb injection.  
**Fix:** Same as BUG-002 and BUG-R04.

### BUG-005: Classic — Verb menu shows English labels after scummtr injection
**Status:** RESOLVED  
**Was:** `PatchVerbLayout` embedded a pre-built `scrp_0022_patched.bin` (English labels, patched coordinates) and wrote it verbatim over SCRP_0022 at runtime, discarding the Swedish labels scummtr had just injected.  
**Fix:** Removed the embedded binary. `PatchVerbLayout` now reads the current SCRP_0022 from the dump (Swedish labels intact), patches only the X/Y coordinate bytes in-memory, and reimports.

---

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
