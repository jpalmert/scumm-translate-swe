# scumm-translate-swe

Swedish fan translation of LucasArts SCUMM engine games.
The translation is done with AI assistance (Claude); this repo contains the
tooling, translation data, and a self-contained patcher.

First target: **The Secret of Monkey Island** — both the Special Edition
(GOG/Steam) and the Classic CD-ROM version (ScummVM).

---

## For end users — applying the patch

> **Warning:** Existing savegames will not load after patching. Start a new game.

You need your own legal copy of *The Secret of Monkey Island*.
One patcher works for both the Special Edition and the Classic CD-ROM version.

> **Note:** Tested with the GOG Special Edition and the Classic CD-ROM version in ScummVM. The Steam version should work but has not been verified.

### How to patch

1. Download the zip for your platform from the
   [v0.1 release](https://github.com/jpalmert/scumm-translate-swe/releases/tag/v0.1).
2. Extract the zip into the same folder as your game files.
3. Run the patcher.

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
internal/
  pak/                      PAK archive reader/writer
  backup/                   .bak backup helper
  classic/                  scummtr wrapper (InjectTranslation)
    assets/                 Embedded scummtr binaries (committed to git)
  charset/                  CHAR block patcher + verb layout
    assets/                 Embedded scummrp binaries (committed to git)
  font/                     SE font lookup table patcher

games/
  monkey1/                  MI1 workspace
    translation/            Swedish translation (committed)
    game/                   User's game files (gitignored)
    gen/                    Extracted assets (gitignored)
    dist/                   Built patcher binaries (gitignored)
  monkey2/                  MI2 workspace (skeleton)
    translation/            Swedish translation (to be created)
    game/                   User's game files (gitignored)
    gen/                    Extracted assets (gitignored)
    dist/                   Built patcher binaries (gitignored)

translation/
  glossary.md               Shared translation glossary

tools/                      Python utilities (see tools/README.md)
scripts/                    Build and extraction scripts
docs/                       Design documents and specs
bin/                        Developer tool binaries (committed)
```

### External dependencies

The following third-party tools are bundled in the repository under `bin/` and `internal/classic/assets/` — no download needed.

| Tool | License | Bundled as | Purpose |
|------|---------|------------|---------|
| [scummtr](https://github.com/dwatteau/scummtr) | MIT | `bin/scummtr-linux`, `bin/scummtr-darwin` and `internal/classic/assets/` (Linux + macOS + Windows) | Extract/inject SCUMM dialog strings. `bin/` is used by developer scripts; `assets/` is embedded in the distributed patchers. |

Go 1.21+ and Python 3 must be installed separately.

If you need to rebuild or upgrade the bundled binaries, see [Refreshing dependencies](#refreshing-dependencies) below.

### One-time setup

Place your game files where the scripts can find them:

```bash
# Special Edition (GOG or Steam):
cp /path/to/Monkey1.pak games/monkey1/game/

# Classic CD-ROM — place files directly:
cp /path/to/MONKEY1.000 /path/to/MONKEY1.001 games/monkey1/game/
# Also accepted: lowercase names and MONKEY.000/001 (the CD naming convention)
```

### Extract assets from the game

```bash
# Place game files in games/monkey1/game/, then:
bash scripts/extract.sh monkey1

# Or from inside the game directory:
cd games/monkey1 && bash ../../scripts/extract.sh
```

This populates `games/monkey1/gen/` (gitignored):

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

Requires `games/monkey1/gen/` to be populated (run `extract.sh` first).

```bash
bash scripts/build.sh monkey1

# Output in games/monkey1/dist/:
#   mi1-translate-linux.zip
#   mi1-translate-macos.zip
#   mi1-translate-windows.zip
```

### Refreshing dependencies

Both Linux and macOS scummtr binaries are bundled. To upgrade to a newer version, run:

```bash
bash scripts/install_deps.sh
```

This re-downloads all tool binaries and updates both `bin/` and the embedded assets in `internal/*/assets/`. Commit the updated binaries afterwards.

### Running tests

```bash
# Unit + Python tests (no game files needed):
bash scripts/test.sh monkey1

# All tests including integration (requires games/monkey1/game/Monkey1.pak):
bash scripts/test.sh monkey1 --all
```

---

## How it works

The GOG and Steam versions of MI1SE store game dialog inside embedded classic SCUMM
resource files (`classic/en/MONKEY1.000` and `classic/en/MONKEY1.001`) within `Monkey1.pak`.
These are identical to the CD-ROM classic version that `scummtr` was built for.

The Swedish translation (`games/monkey1/translation/swedish.txt`) is sourced from the
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

1. Create a new game workspace: `mkdir -p games/<game>/{game,gen,dist,translation}`
2. Find the scummtr game ID for your game (see table below, or run `bin/linux/scummtr -L`).
3. Place game files in `games/<game>/game/` and extract assets:
   ```
   bash scripts/extract.sh <game>
   ```
4. Translate `games/<game>/gen/strings/english.txt` → `games/<game>/translation/swedish.txt`.
5. Add patcher code following the Monkey Island 1 pattern.

### Special Edition games

1. Create a new game workspace under `games/<game>/`.
2. Investigate the PAK structure with `tools/pak.py extract`.
3. Determine whether the game uses embedded classic SCUMM files (use the scummtr approach)
   or SE-specific `.info` text files — investigate as needed.
4. Add translation and patcher code following the existing pattern.

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
