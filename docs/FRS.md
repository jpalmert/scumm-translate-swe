# Functional Requirements Specification
## SCUMM Swedish Fan Translation Toolkit

**Version:** 0.3  
**Status:** Draft  
**Last updated:** 2026-04-05

---

## 1. Purpose

A toolkit and automated workflow for producing a Swedish fan translation of LucasArts SCUMM engine games. Claude performs the translation. The end product is a self-contained patcher that end users run against their own legally-obtained game files.

---

## 2. Scope

### 2.1 In scope
- Text extraction from game resource files
- AI-assisted translation (Claude) with multi-pass review workflow
- Re-injection of translated text
- Font lookup table patching for Swedish characters (Å, Ä, Ö, å, ä, ö, é)
- Self-contained patcher executable for end-user distribution
- Support for Special Edition (SE) release — primary target
- Support for Classic DOS/CD-ROM release via ScummVM — secondary target

### 2.2 Out of scope
- Graphics/image translation
- Voice acting / audio translation
- Engine modifications

### 2.3 Game targets
| Priority | Game | Status |
|----------|------|--------|
| P0 | The Secret of Monkey Island SE (MI1SE) | First implementation |
| P1 | Monkey Island 2 SE (MI2SE) | Follow-up |
| P2 | Other SCUMM games | Future |

MI1 is used as a test bed because an existing Swedish translation (from monkeycd_swe) is available for end-to-end validation without requiring new translation work.

### 2.4 Platform priority
| Platform | Priority | Notes |
|----------|----------|-------|
| MI1SE GOG version | P0 | Primary test target |
| MI1SE Steam version | P1 | Should work; verify after GOG |
| MI1 Classic (ScummVM) | P2 | Low extra effort; include if feasible |

---

## 3. Functional Requirements

### FR-1: Text Extraction
- The system shall extract all translatable text from game resource files: dialogue, object names, UI strings, verb labels, hints
- For SE: extracted from the embedded classic SCUMM files (`MONKEY1.000`/`MONKEY1.001`) inside the `.pak` archive, using `scummtr`
- For Classic: extracted from SCUMM resource files (`.000`/`.001`) via scummtr
- Text file format shall be compatible with the monkeycd_swe project (scummtr format) to allow reuse of existing Swedish translations for testing
- Extraction supports both a PAK file and a directory of pre-extracted classic files as input

### FR-2: Translation Workflow
- The system shall support a multi-pass translation workflow:
  - Pass 1: Initial translation (Claude)
  - Pass 2: Review pass (Claude reviewing its own output)
- Translation shall preserve SCUMM control codes (e.g. `\255\003` for pauses)
- Translation shall warn when a string exceeds the maximum allowed length (256 chars for SE fixed-stride format)
- The workflow shall support partial translation: translate in batches, save progress, resume later

### FR-3: Text Injection
- Translated text shall be re-injected into game resource files
- Swedish special characters shall be mapped to available charset slots:

| Character | Slot |
|-----------|------|
| Å | \091 |
| Ä | \092 |
| Ö | \093 |
| å | \123 |
| ä | \124 |
| ö | \125 |
| é | \130 |

### FR-4: Font Lookup Table Patching
- The SE engine renders characters via a glyph lookup table in `.font` files
- The SE fonts already contain Swedish glyphs at Windows-1252 code positions
- After text injection (which uses SCUMM internal codes), the lookup table must be patched to point each SCUMM code to the correct existing glyph
- This is implemented as a pure lookup-table patch in Go (`internal/font`) — no new glyph images are added
- Classic: no font patching needed (ScummVM handles charset rendering)

### FR-5: Distribution — Self-Contained Patcher
- The end-user deliverable shall be a **self-contained executable patcher**
- The patcher shall locate or prompt for the user's game installation
- The patcher shall apply the translation without requiring the user to install any additional tools
- The patcher shall **not** contain or distribute original game files
- The patcher shall validate the source files (checksum) before patching to confirm it has the right version
- The patcher shall produce clear success/error output

### FR-6: SE Support
- Shall support MI:SE (game=1) and MI2:SE (game=2)
- Shall read and repack the `.pak` archive (both GOG `KAPL` and Steam `LPAK` magic)
- Translated content replaces the English strings in the embedded classic SCUMM files
- No language setting change required — Swedish text is active on a new game

### FR-7: Classic SCUMM Support (secondary)
- Shall support classic SCUMM games playable via ScummVM
- Text extraction and injection via scummtr (embedded in the patcher binary)
- Distribution via self-contained Go binary patcher (`cmd/patcher`)
- Accepts game directory with upper or lowercase filenames

### FR-8: Developer Tooling
- A single setup script shall install all development dependencies (recovery only — binaries are bundled)
- The full extraction → translation → injection → packaging pipeline shall be scriptable end-to-end
- Developer tooling supports Linux and macOS

---

## 4. Non-Functional Requirements

### NFR-1: Legal compliance
- The patcher shall not include or distribute original copyrighted game data
- Only translation data (strings, font lookup table patches) may be distributed

### NFR-2: Version safety
- The patcher shall validate source file checksums before applying changes
- The patcher shall refuse to patch unknown or already-patched files with a clear error message

### NFR-3: Platforms
- Patcher: Windows, Linux, and macOS (self-contained Go binary for all three)
- Dev tooling: Linux (primary, tested); macOS (supported, best-effort)

---

## 5. Constraints

- Swedish text is on average longer than English — string length budget must be tracked
- Savegames created before patching may be incompatible with patched game files (noted separately above)
- GOG and Steam versions may differ in file layout/checksums — primary target is GOG

---

## 6. Open Issues

None. All critical blockers resolved as of 2026-04-05.
