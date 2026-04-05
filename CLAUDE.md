# CLAUDE.md — Session Context

## What this project is

A toolkit for creating a Swedish fan translation of LucasArts SCUMM engine games,
with Claude doing the translation. See `docs/FRS.md` for full requirements.

First target: **The Secret of Monkey Island Special Edition (MI1SE)**, GOG version.
The existing Swedish translation from the `monkeycd_swe` repo will be used for
end-to-end testing before any new translation work is needed.

---

## Project status (as of session handoff 2026-04-04)

### What has been built
- `tools/mise/pak.py` — PAK archive extractor/repacker for MI1SE/MI2SE
- `tools/mise/text.py` — `.info` text extractor/injector (all SE formats)
- `tools/mise/font.py` — `.font` glyph expander for Swedish characters
- `scripts/classic/` — scummtr-based extract/inject/patch scripts for classic SCUMM
- `scripts/se/` — full SE pipeline scripts (extract for translation, build)
- `scripts/install_deps.sh` — one-time dependency installer
- `docs/FRS.md` — Functional Requirements Spec v0.2
- `docs/OPEN_QUESTIONS.md` — 8 open questions, 2 are P0 blockers

### What has NOT been done yet
- Dependencies not installed (`bash scripts/install_deps.sh` not run yet)
- No actual game files have been tested against
- OQ-1 and OQ-2 (see below) are unresolved blockers
- The self-contained end-user patcher has not been built yet
- The graphics directory (`games/monkey1/graphics/`) still exists but is out of scope — delete it

---

## Immediate next steps

1. **Resolve OQ-1**: Get the GOG MI1SE install path. Ask the user to run:
   `find ~ -name "Monkey1.pak" 2>/dev/null`
   Then inspect the file layout with `pak.py extract` to confirm structure.

2. **Resolve OQ-2**: Compare string content between classic scummtr extraction
   and SE `.info` extraction to determine if the monkeycd_swe `text.swe` can be
   used to populate SE translations directly.

3. **Delete graphics directory** (out of scope):
   `rm -rf games/monkey1/graphics`

4. **Install deps** once the user is ready:
   `bash scripts/install_deps.sh`

---

## Key decisions made

- **Swedish only** (not a generic multi-language toolkit)
- **SE is primary target**, classic ScummVM is secondary (low extra effort)
- **No graphics translation** (out of scope)
- **Self-contained patcher** for distribution — user should not need external tools
- **GOG MI1SE is P0** test target (Steam is P1)
- **No GUI** anywhere — Claude does the translation, all tooling is CLI/scriptable
- **French slot limitation**: SE engine only lets us replace one language; we replace
  French. Users must set game language to French. This is unavoidable at engine level.
  Investigate whether we can auto-patch a config file to set language (OQ-4).
- **monkeycd_swe text format compatibility**: translations stored in scummtr format
  so existing Swedish test translations can be reused

---

## P0 open questions (blockers)

**OQ-1 — GOG vs Steam file layout**
Does GOG MI1SE use the same file structure as Steam? `pak.py` was derived from
MISETranslator which was Steam-only. Need to verify PAK layout, file locations,
and `.info` format versions against a real GOG install before writing any patcher.

**OQ-2 — scummtr format ↔ SE .info string alignment**
The existing Swedish translation (`monkeycd_swe/src/TEXT/text.swe`) is in scummtr
format extracted from classic MONKEY.000/001. The SE stores text in `.info` files.
Do string IDs align? Can we convert directly, or is a mapping/reconciliation step needed?
This determines whether we can test end-to-end without doing new translation work.

See `docs/OPEN_QUESTIONS.md` for all 8 open questions.

---

## Repo structure
```
docs/               FRS.md, OPEN_QUESTIONS.md
games/monkey1/
  text/             Classic text files (scummtr format)
  se_translations/  SE JSON files (english + translation fields)
  references/       TRANSLATE_TABLE (Swedish char code mappings)
  patches/          Output patch files
scripts/
  install_deps.sh
  classic/          extract_text.sh, inject_text.sh
  se/               extract_for_translation.sh, build.sh
tools/
  bin/              Built tool binaries (gitignored, created by install_deps.sh)
  mise/             pak.py, text.py, font.py + README.md
.claude/
  settings.json     Project permissions (allow all tools)
```

## Reference: monkeycd_swe
The reference project is at `~/monkeycd_swe`. It contains:
- `src/TEXT/text.swe` — complete Swedish translation in scummtr format (~4400 lines)
- `src/REFERENCES/TRANSLATE_TABLE` — Swedish character code mappings
- `patches/` — working BPS patches for classic MI1 CD version
This is our test data source.

## Memory
Full context is in `~/.claude/projects/-home-jpalmert-scumm-translation/memory/`.
Key files: workflow_classic.md, workflow_se.md, file_formats_se.md, tools_reference.md, custom_tools.md.
