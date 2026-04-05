# Functional Requirements Specification
## SCUMM Swedish Fan Translation Toolkit

**Version:** 0.2  
**Status:** Draft  
**Last updated:** 2026-04-04

---

## 1. Purpose

A toolkit and automated workflow for producing a Swedish fan translation of LucasArts SCUMM engine games. Claude performs the translation. The end product is a self-contained patcher that end users run against their own legally-obtained game files.

---

## 2. Scope

### 2.1 In scope
- Text extraction from game resource files
- AI-assisted translation (Claude) with multi-pass review workflow
- Re-injection of translated text
- Font expansion for Swedish characters (Å, Ä, Ö, å, ä, ö, é)
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
- For SE: extracted from `.info` files inside the `.pak` archive
- For Classic: extracted from SCUMM resource files (`.000`/`.001`) via scummtr
- Text file format shall be compatible with the monkeycd_swe project (scummtr format) to allow reuse of existing Swedish translations for testing

### FR-2: Translation Workflow
- The system shall support a multi-pass translation workflow:
  - Pass 1: Initial translation (Claude)
  - Pass 2: Review pass (Claude reviewing its own output)
  - *(Further passes TBD — see docs/OPEN_QUESTIONS.md)*
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

### FR-4: Font Expansion
- The system shall expand the game font to include Swedish diacritical characters at the slots defined in FR-3
- SE: via tools/mise/font.py
- Classic: via scummfont + modified charset BMP

### FR-5: Distribution — Self-Contained Patcher
- The end-user deliverable shall be a **self-contained executable patcher**
- The patcher shall locate or prompt for the user's game installation
- The patcher shall apply the translation without requiring the user to install any additional tools
- The patcher shall **not** contain or distribute original game files
- The patcher shall validate the source files (checksum) before patching to confirm it has the right version
- The patcher shall produce clear success/error output

### FR-6: SE Support
- Shall support MI:SE (game=1) and MI2:SE (game=2)
- Shall extract/repack the `.pak` archive
- Translated content replaces the French language slot (engine limitation)
- End user must set game language to French to see the Swedish translation
- *(This UX issue is noted as a known limitation — see docs/OPEN_QUESTIONS.md)*

### FR-7: Classic SCUMM Support (secondary)
- Shall support classic SCUMM games playable via ScummVM
- Text extraction and injection via scummtr
- Distribution via BPS patch files (applied with Floating IPS or equivalent)

### FR-8: Developer Tooling
- A single setup script shall install all development dependencies
- The full extraction → translation → injection → packaging pipeline shall be scriptable end-to-end

---

## 4. Non-Functional Requirements

### NFR-1: Legal compliance
- The patcher shall not include or distribute original copyrighted game data
- Only translation data (strings, font glyphs) may be distributed

### NFR-2: Version safety
- The patcher shall validate source file checksums before applying changes
- The patcher shall refuse to patch unknown or already-patched files with a clear error message

### NFR-3: Platforms
- Patcher: Windows primary (largest user base for GOG game installs); Linux/macOS secondary
- Dev tooling: Linux (developer machine)

---

## 5. Constraints

- Swedish text is on average longer than English — string length budget must be tracked
- SE engine limitation: only the French language slot can be replaced; game must be set to French
- Savegames created before patching may be incompatible with patched game files
- GOG and Steam versions may differ in file layout/checksums — primary target is GOG

---

## 6. Open Issues

See `docs/OPEN_QUESTIONS.md` for a full list of unresolved questions.

Critical blockers — all resolved as of 2026-04-05:
- OQ-1: GOG vs Steam file layout differences — RESOLVED
- OQ-2: String ID alignment between scummtr format and SE .info format — RESOLVED
- OQ-3: Self-contained patcher technology choice — RESOLVED (Go binary)
