# CLAUDE.md — Session Context

## What this project is

A self-contained patcher that injects a Swedish translation into LucasArts SCUMM games.
Claude does the translation work; this repo contains the tooling and translation data.

First target: **Monkey Island 1** — both the Special Edition (GOG/Steam `Monkey1.pak`) and the
Classic CD-ROM version (`MONKEY1.000`/`MONKEY1.001` via ScummVM).

The Swedish translation (`translation/monkey1/swedish.txt`) is sourced from the
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

translation/
  monkey1/
    swedish.txt                 Swedish translation (4437 strings, scummtr format)
    TRANSLATE_TABLE             Swedish character code mappings
    glossary.md                 Translation decisions reference
    PASS1_NOTES.md              Insult swordfighting translation notes

docs/
  FRS.md                        Functional requirements
  TEST_PLAN.md                  Test plan (unit, integration, manual)
  TRANSLATION_PLAN.md           Multi-pass translation workflow
  TRANSLATION_GUIDE.md          String format, opcodes, control codes, encoding

tools/                          Python utilities for PAK inspection (not part of build pipeline)
  pak.py                        PAK extractor/repacker
  patch_verbs.py                Verb button coordinate patcher (standalone inspection)

scripts/
  extract.sh                    Entry point: detect PAK/dir, call sub-scripts
  extract_pak.sh                Unpack MONKEY1.000/001 from SE PAK → game/monkey1/
  extract_assets.sh             Extract CHAR blocks, BMPs, dialog strings from game dir
  build.sh                      Generate CHAR assets + cross-compile patcher → dist/
  clean.sh                      Remove generated .bin files and dist/ binaries
  clean_assets.sh               Remove all assets extracted from the game
  install_deps.sh               Re-download tool binaries (needed only for upgrades)

bin/
  linux/                        Developer tool binaries (scummtr, scummrp, scummfont, FontXY) — committed
  darwin/                       Same for macOS

--- gitignored ---

game/monkey1/                   User's game files (never commit copyrighted content)
  Monkey1.pak                   Place SE PAK here (or pass path to extract.sh)
  MONKEY1.000 / MONKEY1.001     Classic files (or unpacked from PAK by extract_pak.sh)
  gen/                          All assets extracted from game (regenerate with extract.sh)
    charset/english/            Raw CHAR blocks (templates for build.sh)
    charset/english_bitmaps/    English glyph BMPs (visual reference)
    strings/english.txt         English dialog strings for translation

internal/charset/gen/           Generated CHAR .bin files (run scripts/build.sh to populate)
dist/                           Built patcher binaries
```

---

## Core pipeline

### Setup (once)

```bash
# Tool binaries are committed to bin/ and internal/*/assets/ — nothing to install.
# Only needed if upgrading scummtr/scummrp or if binaries are corrupted:
bash scripts/install_deps.sh
```

### Extract game assets

```bash
# Place game files in game/monkey1/, then:
bash scripts/extract.sh                          # auto-detects PAK vs classic files
bash scripts/extract.sh /path/to/Monkey1.pak    # explicit PAK path
bash scripts/extract.sh /path/to/game/dir/      # explicit game dir
```

This populates `game/monkey1/gen/` with CHAR blocks, BMPs, and English dialog strings.

### Build the patcher

```bash
# Requires Go 1.21+ and extracted game assets.
bash scripts/build.sh
# Output: dist/mi1-translate-linux, dist/mi1-translate-darwin, dist/mi1-translate-windows.exe, dist/swedish.txt
```

### Run tests

```bash
go test ./...                            # unit tests (fast, no game files needed)
go test -tags integration ./...         # unit + integration (needs game/monkey1/Monkey1.pak)
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

---

## Reference: monkeycd_swe

At `~/monkeycd_swe`:
- `src/TEXT/text.swe` — complete Swedish translation in scummtr format (~4400 lines)
- `src/REFERENCES/TRANSLATE_TABLE` — Swedish character code mappings
- `patches/` — working BPS patches for classic MI1 CD version

---

## Memory

Full context is in `~/.claude/projects/-home-jpalmert-scumm-translation/memory/`.
