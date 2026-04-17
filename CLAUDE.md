# CLAUDE.md — Session Context

## What this project is

A self-contained patcher that injects a Swedish translation into LucasArts SCUMM games.
Claude does the translation work; this repo contains the tooling and translation data.

First target: **Monkey Island 1** — both the Special Edition (GOG/Steam `Monkey1.pak`) and the
Classic CD-ROM version (`MONKEY1.000`/`MONKEY1.001` via ScummVM).

The Swedish translation (`games/monkey1/translation/swedish.txt`) is sourced from the
[monkeycd_swe](https://github.com/dwatteau/monkeycd_swe) project and aligns 1:1 with
the SE strings (4437 strings in the same order). Insult swordfighting strings have been
translated (see `docs/TRANSLATION_PLAN.md` for the multi-pass translation workflow).

---

## Repo structure

```
go.mod                          Go module (scumm-patcher)
cmd/
  patcher/                      Single binary: auto-detects SE vs Classic, patches game files
    main.go                     Entry point, auto-detection, arg parsing
    se.go                       SE pipeline (PAK read/repack, classic inject, font remap, speech sync)
    classic.go                  Classic pipeline (backup, scummtr inject, CHAR patch, verb layout)
    patch.go                    Shared patchClassicFiles helper
    translation.go              Translation file lookup
    patcher_test.go             Unit tests (SE and Classic error paths)
    integration_test.go         Integration tests (//go:build integration, needs Monkey1.pak)

internal/
  pak/                          PAK archive reader/writer (KAPL/LPAK)
  backup/                       .bak safety copy helper
  classic/                      scummtr wrapper (InjectTranslation, BuildSpeechMapping)
    assets/                     Embedded scummtr binaries (Linux/macOS/Windows) — committed to git
  charset/                      CHAR block patcher (Swedish glyphs) + verb layout patcher
    assets/                     Embedded scummrp binaries — committed to git
    bitmaps/                    Swedish glyph BMP source files — committed to git
    gen/                        Generated .bin files — gitignored; run scripts/build.sh to populate
  font/                         SE .font glyph lookup table patcher
  speech/                       speech.info audio sync patcher

games/
  monkey1/
    translation/                Committed — MI1 translation data
      swedish.txt               Swedish translation (4437 strings, scummtr format)
      TRANSLATE_TABLE           Swedish character code mappings
      PASS1_NOTES.md            Insult swordfighting translation notes
      annotations.md            Per-string translation annotations
    game/                       Gitignored — user's MI1 game files
    gen/                        Gitignored — extracted resources
    dist/                       Gitignored — built MI1 patcher binaries
  monkey2/
    translation/                Committed — MI2 translation data (to be created)
    game/                       Gitignored — user's MI2 game files
    gen/                        Gitignored — extracted resources
    dist/                       Gitignored — built MI2 patcher binaries

translation/
  glossary.md                   Shared translation glossary (used by all games)

docs/
  FRS.md                        Functional requirements
  TEST_PLAN.md                  Test plan (unit, integration, manual)
  TRANSLATION_PLAN.md           Multi-pass translation workflow
  TRANSLATION_GUIDE.md          String format, opcodes, control codes, encoding

tools/                          Python utilities (see tools/README.md for usage)
  decode_room.py                Decode SCUMM v5 room backgrounds → PNG (called by extract_assets.sh)
  decode_object.py              Decode SCUMM v5 object images → PNG (called by extract_assets.sh)
  find_dynamic_names.py         Extract runtime name-change mapping from SCUMM scripts (standalone)
  calc_padding.py               Apply @ padding to swedish.txt for SE name buffers (called by build.sh)
  pak.py                        PAK extractor/repacker (standalone)
  patch_verbs.py                Verb button coordinate patcher (standalone)
  scumm_gfx.py                  Shared SCUMM v5 graphics codec library

scripts/
  common.sh                     Shared helpers (REPO_ROOT, detect_game from pwd)
  init_translation.sh           Init swedish.txt with [E]-prefixed English strings (first-time setup)
  extract.sh                    Entry point: detect PAK/dir, call sub-scripts
  extract_pak.sh                Unpack MONKEY1.000/001 from SE PAK → games/<game>/game/
  extract_assets.sh             Extract CHAR blocks, BMPs, dialog strings, room/object images
  build.sh                      Generate CHAR assets + cross-compile patcher → games/<game>/dist/
  clean.sh                      Remove generated .bin files and dist/ binaries
  clean_assets.sh               Remove all assets extracted from the game
  install_deps.sh               Re-download tool binaries (needed only for upgrades)

bin/
  linux/                        Developer tool binaries (scummtr, scummrp, scummfont, FontXY, descumm) — committed
  darwin/                       Developer tool binaries (scummtr, scummrp, scummfont, FontXY) — committed
```

---

## Core pipeline

Scripts detect the active game from the working directory. Run them from inside
`games/<game>/` (e.g. `cd games/monkey1`).

### Setup (once)

```bash
# Tool binaries are committed to bin/ and internal/*/assets/ — nothing to install.
# Only needed if upgrading scummtr/scummrp or if binaries are corrupted:
bash scripts/install_deps.sh
```

### Extract game assets

```bash
# Place game files in games/monkey1/game/, then:
cd games/monkey1
bash ../../scripts/extract.sh                          # auto-detects PAK vs classic files
bash ../../scripts/extract.sh /path/to/Monkey1.pak    # explicit PAK path
bash ../../scripts/extract.sh /path/to/game/dir/      # explicit game dir
```

This populates `games/monkey1/gen/` with CHAR blocks, BMPs, English dialog strings,
room background PNGs, and object image PNGs.

### Build the patcher

```bash
# Requires Go 1.21+ and extracted game assets.
cd games/monkey1
bash ../../scripts/build.sh
# Output: games/monkey1/dist/mi1-translate-linux, .../mi1-translate-darwin, .../mi1-translate-windows.exe, .../swedish.txt
```

The build copies `swedish.txt` to `dist/` and automatically applies `@` padding to
object names that have runtime replacements (e.g. "mug" → "mug of grog"). This
prevents buffer overflows in the SE engine which writes replacement names in-place.
The source `swedish.txt` is never modified.

### Translation workflow

**Starting a new game translation (first time only):**

```bash
# 1. Extract game assets (populates games/<game>/gen/strings/english.txt)
cd games/monkey1
bash ../../scripts/extract.sh

# 2. Initialise the translation file — writes [E]-prefixed English lines into
#    games/<game>/translation/swedish.txt. Safe: refuses to overwrite existing work.
bash ../../scripts/init_translation.sh
```

The `[E]` prefix marks untranslated lines. A translated line looks like:
```
# Before:  [001:OBNA#0016][E][001:OBNA#0016]jungle
# After:   [001:OBNA#0016]djungel
```
Remove the `[E]` prefix as you translate each line.

**Ongoing translation (per-session with Claude):**

1. Open `games/monkey1/translation/swedish.txt` — `[E]`-prefixed lines are untranslated.
2. Translate in Claude, following `docs/TRANSLATION_GUIDE.md` for format rules
   and `translation/glossary.md` for vocabulary decisions.
3. Replace `[E]`-prefixed lines with the Swedish translation (no prefix).
4. Build and test: `cd games/monkey1 && bash ../../scripts/build.sh`, then run the patcher on game files.

**Translation reference docs:**
- `docs/TRANSLATION_GUIDE.md` — file format, opcodes, control codes
- `docs/TRANSLATION_PLAN.md` — multi-pass workflow and translation philosophy
- `translation/glossary.md` — vocabulary and naming decisions

---

### Run tests

```bash
go test ./...                            # unit tests (fast, no game files needed)
go test -tags integration ./...         # unit + integration (needs games/monkey1/game/Monkey1.pak)
```

**Note:** `charset` asset tests (`-tags buildpatcher`) validate the embedded CHAR blocks and only
apply when building the full patcher. Run them after `build.sh` Step 2 if you've edited glyphs:
```bash
go test -tags buildpatcher ./internal/charset/...
```

---

## Key decisions

- **Swedish only** — not a generic multi-language toolkit
- **SE is primary target** — Classic ScummVM is secondary (low extra effort)
- **No graphics translation** — all text in graphics is proper nouns that stay in English
- **Self-contained patcher** — single binary + `swedish.txt`; user needs no other tools
- **English strings replaced directly** — no language setting change required
- **scummtr format** — `swedish.txt` uses scummtr format, compatible with monkeycd_swe
- **Multi-game structure** — each game has its own workspace under `games/<game>/`

---

## Documents

Read these when working on translation or tooling:

| File | Read when... |
|------|-------------|
| `docs/TRANSLATION_GUIDE.md` | Translating strings — format, opcodes, control codes, Swedish encoding |
| `docs/TRANSLATION_PLAN.md` | Starting or continuing a translation pass — workflow, philosophy, length rules |
| `docs/FRS.md` | Checking functional requirements or adding a new game |
| `docs/TEST_PLAN.md` | Writing or running tests — what exists and what it covers |
| `docs/INSULT_COMEBACK_MAPPINGS_ENGLISH.md` | Working on sword-fight insults/comebacks — the EN pairs and their logic |
| `translation/glossary.md` | Any translation decision — vocabulary, names, register choices |
| `games/monkey1/translation/PASS1_NOTES.md` | Reviewing or continuing insult swordfighting translations |
| `tools/README.md` | Using or modifying the Python tools |

---

## Working conventions

These apply to all work in this repo:

**Tests:** Always write a test for new functionality. Always run `go test ./...` after changing existing code. Tests are part of completing a task, not a separate step.

**Commits:** Commit to git whenever a unit of work is complete and working — don't let changes accumulate. Use `git add <specific files>` not `git add -A` to avoid accidentally staging game files or build artifacts.

**Gitignored files:** Never use `git add -f` to force-commit files from `games/*/game/`, `games/*/gen/`, `games/*/dist/`, or other gitignored paths. Those directories contain game files and build outputs that must not be in the repo.

**Embedded binaries:** Never create placeholder files for `//go:embed` assets. If a binary is missing, download or build the real one immediately. A placeholder compiles but fails at runtime.

**File deletion:** Delete files individually (`rm file1 file2`). `rm -rf` is blocked by the sandbox; use `rmdir` for empty directories after removing their contents.

---

## Reference: monkeycd_swe

At `~/monkeycd_swe`:
- `src/TEXT/text.swe` — complete Swedish translation in scummtr format (~4400 lines)
- `src/REFERENCES/TRANSLATE_TABLE` — Swedish character code mappings
- `patches/` — working BPS patches for classic MI1 CD version

---

## Memory

Full context is in `~/.claude/projects/-home-jpalmert-scumm-translation/memory/`.
