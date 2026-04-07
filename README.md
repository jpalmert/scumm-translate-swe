# scumm-translation

A toolkit for creating a Swedish fan translation of LucasArts SCUMM engine games,
with AI assistance (Claude) doing the translation work.

First target: **The Secret of Monkey Island Special Edition (MI1SE)**, GOG version.

---

## For end users — applying the patch

> **Warning:** Existing savegames will not load after patching. Start a new game.

You need your own legal copy of *The Secret of Monkey Island*.
One patcher works for both the Special Edition and the Classic CD-ROM version.

> **Note:** Only tested with the GOG Special Edition. The Steam version should work but has not been verified.

### How to patch

Download `mi1-translate-windows.exe` / `mi1-translate-darwin` / `mi1-translate-linux`
and `swedish.txt` into the same folder as your game files, then run the patcher.

The patcher detects your version automatically:
- **Special Edition:** place next to `Monkey1.pak`
- **Classic CD-ROM:** place next to `MONKEY1.000` and `MONKEY1.001`

**Windows:**
```
mi1-translate-windows.exe
```

**Linux / macOS:**
```
./mi1-translate-linux
```

A backup of your game files is created automatically before any changes are made.

After patching, start a new game. The Swedish text replaces the English strings directly.

**Advanced:** you can also pass paths explicitly:
```
mi1-translate-linux Monkey1.pak [output.pak] [swedish.txt]   # SE
mi1-translate-linux /path/to/game/dir [swedish.txt]          # Classic
```


---

## For developers — building and extending the toolkit

> **Platform support:** The developer toolchain (scripts, Go build, Python tools) is designed to work on both Linux and macOS. Only Linux has been tested. macOS support is best-effort — binaries for both platforms are bundled, and platform-specific differences (like `sed` behaviour) have been accounted for, but macOS paths have not been exercised end-to-end.

### Repository structure

```
go.mod                      Go module (module scumm-patcher)
cmd/
  patcher/                  Single CLI: auto-detects SE vs Classic, patches game files
    main.go                 Entry point, auto-detection, dispatch
    se.go                   SE pipeline (PAK read/repack, scummtr inject, font remap)
    classic.go              Classic pipeline (find files, backup, scummtr inject)
    translation.go          Shared translation file lookup
    patcher_test.go         Unit tests
    integration_test.go     Integration tests (requires Monkey1.pak)
internal/
  pak/                      PAK archive reader/writer
    pak.go
    pak_test.go
  backup/                   .bak backup helper
    backup.go
    backup_test.go
  classic/                  scummtr wrapper (InjectTranslation)
    classic.go
    embed.go                go:embed for scummtr binaries
    classic_integration_test.go
    assets/                 Embedded scummtr binaries (committed to git)
      scummtr-linux-x64
      scummtr-darwin-x64
      scummtr-windows-x64.exe
  font/                     SE font lookup table patcher
    font.go
    font_test.go

tools/                      Python tools for SE translation work (developer use)
  pak.py                    PAK archive extractor/repacker
  text.py                   .info file text extractor/injector
  README.md

scripts/
  extract.sh                Top-level entry point: detects PAK vs game dir, calls sub-scripts
  extract_pak.sh            Unpack MONKEY1.000/001 from an SE PAK archive → game/monkey1/
  extract_assets.sh         Extract CHAR blocks, BMPs and dialog strings from a game dir
  extract_text.sh           Generic scummtr text extraction (any SCUMM game/game-ID)
  build.sh                  Generate Swedish charset assets and cross-compile the patcher
  clean.sh                  Remove build artifacts (gen/ .bin files and dist/ binaries)
  clean_assets.sh           Remove all assets extracted from the game (undoes extract.sh)
  install_deps.sh           Re-download tool binaries if the committed copies are damaged

translation/
  monkey1/
    swedish.txt             Swedish translation (scummtr format, 4437 strings)
    TRANSLATE_TABLE         Swedish character code mappings

docs/
  FRS.md                   Functional Requirements Spec
  TEST_PLAN.md             Test plan

--- gitignored below this line ---

game/                       User-provided copyrighted game files (never committed)
  monkey1/
    Monkey1.pak             SE PAK file (GOG or Steam) — place here or pass path to extract.sh
    MONKEY1.000             Classic files — either place here directly or unpacked from PAK
    MONKEY1.001
    gen/                    All assets extracted from the game (regenerate with extract.sh)
      charset/english/      Raw CHAR font blocks (used by build.sh)
      charset/english_bitmaps/ English reference BMPs (visual aid for editing Swedish glyphs)
      strings/english.txt   English dialog strings for translation

bin/                        Downloaded tool binaries (never committed)

dist/                       Built patcher binaries (never committed)
  mi1-translate-linux
  mi1-translate-darwin
  mi1-translate-windows.exe
  swedish.txt               ← shipped alongside the binary
```

### External dependencies

The following third-party tools are bundled in the repository under `bin/` and `internal/classic/assets/` — no download needed.

| Tool | License | Bundled as | Purpose |
|------|---------|------------|---------|
| [scummtr](https://github.com/dwatteau/scummtr) | MIT | `bin/scummtr-linux`, `bin/scummtr-darwin` and `internal/classic/assets/` (Linux + macOS + Windows) | Extract/inject SCUMM dialog strings. `bin/` is used by developer scripts; `assets/` is embedded in the distributed patchers. |

Go 1.21+ and Python 3 must be installed separately.

If you need to rebuild or upgrade the bundled binaries, see [Refreshing dependencies](#refreshing-dependencies) below.

### One-time setup

Place your game files where the scripts can find them. Either the SE PAK or the
classic SCUMM files work — the scripts detect which you have:

```bash
# Special Edition (GOG or Steam) — default location:
cp /path/to/Monkey1.pak game/monkey1/

# Classic CD-ROM — place files directly:
cp /path/to/MONKEY1.000 /path/to/MONKEY1.001 game/monkey1/
# Also accepted: lowercase names and MONKEY.000/001 (the CD naming convention)
```

### Extract assets from the game

Run this once after placing your game files. It extracts the font data and English dialog
strings needed for building the patcher and for translation work.

```bash
# Auto-detects PAK vs classic files (uses game/monkey1/ by default):
bash scripts/extract.sh

# Explicit paths:
bash scripts/extract.sh /path/to/Monkey1.pak
bash scripts/extract.sh /path/to/game/dir/
```

This populates `game/monkey1/gen/` (gitignored):

| Output | Purpose |
|--------|---------|
| `gen/charset/english/CHAR_NNNN` | Raw CHAR font blocks — templates for `build.sh` |
| `gen/charset/english_bitmaps/*.bmp` | English glyphs as BMPs — visual reference when editing Swedish glyphs in `internal/charset/bitmaps/` |
| `gen/strings/english.txt` | English dialog strings for translation |

`gen/strings/english.txt` is UTF-8, one string per line, with a `[room:TYPE#resnum](opcode)` prefix:

```
[028:LSCR#0220](D8)Ahh, I'm finally going to be a pirate!
[028:LSCR#0220](14)What do you want?
[028:LSCR#0220](54)rusty sword
```

The opcode indicates who is speaking: `(D8)` = Guybrush, `(14)` = NPC, `(54)` = object name, etc.
See `docs/TRANSLATION_GUIDE.md` for the full opcode reference and control code documentation.

Translate each string in place, keeping the prefix and all `\255\NNN` control codes unchanged.

### Build the distributable patcher

Requires `game/monkey1/gen/` to be populated (run `extract.sh` first).

```bash
# Requires: Go 1.21+
bash scripts/build.sh

# Output:
#   dist/mi1-translate-linux
#   dist/mi1-translate-darwin
#   dist/mi1-translate-windows.exe
#   dist/swedish.txt
```

### Refreshing dependencies

Both Linux and macOS scummtr binaries are bundled. To upgrade to a newer version, run:

```bash
bash scripts/install_deps.sh
```

This re-downloads all tool binaries and updates both `bin/` and the embedded assets in `internal/*/assets/`. Commit the updated binaries afterwards.

### Running tests

```bash
# Unit tests (fast, no game files needed):
go test ./...

# Integration tests (requires game/monkey1/Monkey1.pak):
go test -tags integration -v ./...
```

---

## How it works

The GOG and Steam versions of MI1SE store game dialog inside embedded classic SCUMM
resource files (`classic/en/MONKEY1.000` and `classic/en/MONKEY1.001`) within `Monkey1.pak`.
These are identical to the CD-ROM classic version that `scummtr` was built for.

The Swedish translation (`translation/monkey1/swedish.txt`) is sourced from the
[monkeycd_swe](https://github.com/dwatteau/monkeycd_swe) project. It aligns 1:1 with
the SE strings (4437 strings in the same order).

**SE patcher pipeline:**
1. Reads `Monkey1.pak` (handles both GOG `KAPL` and Steam `LPAK` magic)
2. Creates `Monkey1.pak.bak` (in-place mode)
3. Extracts `MONKEY1.000` + `MONKEY1.001` to a temp directory
4. Runs the embedded `scummtr` to inject Swedish strings into the temp copies
5. Repacks all PAK entries (with modified classic files) into the output PAK

**Classic patcher pipeline:**
1. Finds `MONKEY1.000` + `MONKEY1.001` in the game directory (upper or lowercase)
2. Creates `MONKEY1.000.bak` and `MONKEY1.001.bak`
3. Copies files to a temp directory with uppercase names (scummtr requirement)
4. Runs the embedded `scummtr` to inject Swedish strings
5. Writes the patched files back to their original paths

The Swedish translation replaces the English strings in place — no language setting change required.

---

## Adding support for a new game

### Classic-only games (CD-ROM / ScummVM)

If you have access to the classic game files only (e.g. `MONKEY1.000` / `MONKEY1.001`),
the classic workflow works standalone — no Special Edition required:

1. Find the scummtr game ID for your game (see table below, or run `bin/linux/scummtr -L`).
2. Extract English assets (pass the game directory directly to skip PAK unpacking):
   ```
   bash scripts/extract.sh /path/to/game/
   ```
3. For a generic text-only extraction to a custom output file:
   ```
   bash scripts/extract_text.sh <game_id> /path/to/game/ translation/<game>/text.txt
   ```
4. Translate `translation/<game>/text.txt`.
5. Add a translation directory under `translation/<game>/` and a new patcher command under `cmd/` following the Monkey Island 1 pattern.

The distributable end-user patcher patches game files in-place and works on Windows,
macOS, and Linux — no external tools needed by the end user.

### Special Edition games

1. Investigate the PAK structure with `tools/pak.py extract`.
2. Determine whether the game uses embedded classic SCUMM files (use the scummtr approach)
   or SE-specific `.info` text files (use `tools/text.py` approach).
4. Add a new translation directory under `translation/<game>/`.
5. Add a new command under `cmd/` following the existing pattern.

## scummtr game IDs

| Game | ID |
|------|----|
| Monkey Island 1 SE / GOG embedded classic | `monkeycdalt` |
| Monkey Island 1 CD (MONKEY.000) | `monkeycd` |
| Monkey Island 2 | `monkey2` |
| Day of the Tentacle | `tentacle` |
| Sam & Max | `samnmax` |
| Full Throttle | `ft` |
| The Dig | `dig` |

Run `bin/scummtr -L` for the full list.
