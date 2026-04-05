# scumm-translation

A toolkit for creating a Swedish fan translation of LucasArts SCUMM engine games,
with AI assistance (Claude) doing the translation work.

First target: **The Secret of Monkey Island Special Edition (MI1SE)**, GOG version.

---

## For end users — applying the patch

> **Warning:** Existing savegames will not load after patching. Start a new game.

You need your own legal copy of *The Secret of Monkey Island*.
Two patchers are provided depending on which version you own.

### Special Edition (GOG or Steam)

Download: `se-patcher-windows.exe` / `se-patcher-darwin` / `se-patcher-linux`
and `monkey1_swe.txt` — keep them in the same folder.

```
se-patcher-windows.exe  "C:\Games\Monkey Island\Monkey1.pak"
```

This patches `Monkey1.pak` in-place. A backup (`Monkey1.pak.bak`) is created
automatically before any changes are made.

After patching, **set the in-game language to French** — the SE engine stores custom
translations in the French language slot. You must select French to see Swedish text.

You can edit `monkey1_swe.txt` before running the patcher to adjust the translation.
Place it next to the patcher binary, or pass its path explicitly:

```
se-patcher-linux Monkey1.pak [output.pak] [monkey1_swe.txt]
```

### Classic / CD-ROM (ScummVM)

Download: `classic-patcher-windows.exe` / `classic-patcher-darwin` / `classic-patcher-linux`
and `monkey1_swe.txt` — keep them in the same folder.

```
classic-patcher-linux  /path/to/game/dir
```

`game/dir` is the folder containing `MONKEY1.000` and `MONKEY1.001` (uppercase or lowercase
both work). Backups (`MONKEY1.000.bak`, `MONKEY1.001.bak`) are created before patching.

After patching, open ScummVM, select the game, and **set the game language to French**.

```
classic-patcher-linux game/dir [monkey1_swe.txt]
```


---

## For developers — building and extending the toolkit

> **Platform support:** The developer toolchain (scripts, Go build, Python tools) is designed to work on both Linux and macOS. Only Linux has been tested. macOS support is best-effort — binaries for both platforms are bundled, and platform-specific differences (like `sed` behaviour) have been accounted for, but macOS paths have not been exercised end-to-end.

### Repository structure

```
go.mod                      Go module (module scumm-patcher)
cmd/
  classic-patcher/          CLI: patches MONKEY1.000/.001 in-place
    main.go
    main_test.go
  se-patcher/               CLI: patches Monkey1.pak (reads → injects → repacks)
    main.go
    main_test.go
    integration_test.go
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
  install_deps.sh           Recovery only — re-downloads scummtr if bundled binaries don't work
  build_patcher.sh          Build distributable binaries for all platforms
  clean.sh                  Remove generated dist/ binaries and scummtr assets
  se/
    extract_classic_strings.sh   Extract English dialog from SE PAK (developer use)
    extract_for_translation.sh   Extract SE text files to JSON (future games)
    build.sh                     Inject JSON translations into SE PAK (future games)
  classic/
    extract_text.sh         Extract classic SCUMM text via scummtr
    inject_text.sh          Inject translated text via scummtr

translation/
  monkey1/
    monkey1_swe.txt         Swedish translation (scummtr format, 4437 strings)
    TRANSLATE_TABLE         Swedish character code mappings

docs/
  FRS.md                   Functional Requirements Spec
  OPEN_QUESTIONS.md        Investigation log
  TEST_PLAN.md             Test plan

--- gitignored below this line ---

game/                      User-provided copyrighted game files (never committed)
  monkey1/Monkey1.pak
  monkey1/text/se_english.txt   English strings extracted by extract_classic_strings.sh

bin/                       Downloaded tool binaries (never committed)

dist/                      Built patcher binaries (never committed)
  classic-patcher-linux
  classic-patcher-darwin
  classic-patcher-windows.exe
  se-patcher-linux
  se-patcher-darwin
  se-patcher-windows.exe
  monkey1_swe.txt          ← shipped alongside the binaries
```

### External dependencies

The following third-party tools are bundled in the repository under `bin/` and `internal/classic/assets/` — no download needed.

| Tool | License | Bundled as | Purpose |
|------|---------|------------|---------|
| [scummtr](https://github.com/dwatteau/scummtr) | MIT | `bin/scummtr-linux`, `bin/scummtr-darwin` and `internal/classic/assets/` (Linux + macOS + Windows) | Extract/inject SCUMM dialog strings. `bin/` is used by developer scripts; `assets/` is embedded in the distributed patchers. |

Go 1.21+ and Python 3 must be installed separately.

If you need to rebuild or upgrade the bundled binaries, see [Refreshing dependencies](#refreshing-dependencies) below.

### One-time setup

Copy your game files into `game/monkey1/`. Either the SE PAK or the classic files work:

```bash
# Special Edition (GOG or Steam):
cp /path/to/Monkey1.pak game/monkey1/

# Classic CD-ROM:
cp /path/to/MONKEY1.000 /path/to/MONKEY1.001 game/monkey1/
```

### Extract English strings

This is the starting point for translation work. The script extracts all English dialog
strings into a text file that you then translate line by line.

The script accepts either the SE PAK file or a directory containing the classic files directly:

```bash
# From the SE PAK (default):
bash scripts/se/extract_classic_strings.sh
bash scripts/se/extract_classic_strings.sh /path/to/Monkey1.pak

# From classic files (MONKEY1.000 + MONKEY1.001), e.g. from the CD-ROM version
# or manually extracted from the PAK:
bash scripts/se/extract_classic_strings.sh /path/to/classic/files/

# Output: game/monkey1/text/se_english.txt  (gitignored)
```

The script writes one string per entry with `[room:type#id]` context headers:

```
[0037:0000#0000]
Ahh, I'm finally going to be a pirate!
[0037:0000#0001]
I wonder what's out there beyond the horizon.
```

Translate each string in place, keeping the `[room:type#id]` headers and the file
structure intact. The translated file is then passed to the SE patcher pipeline.

The file uses Windows-1252 encoding with CRLF line endings (scummtr's native format).
It is gitignored and must be regenerated from your own copy of the game.

### Build the distributable patchers

```bash
# Requires: Go 1.21+, curl, unzip
bash scripts/build_patcher.sh

# Output:
#   dist/classic-patcher-linux/darwin/windows.exe
#   dist/se-patcher-linux/darwin/windows.exe
#   dist/monkey1_swe.txt
```

### Refreshing dependencies

Both Linux and macOS scummtr binaries are bundled. To upgrade to a newer version, run:

```bash
bash scripts/install_deps.sh
```

This re-downloads scummtr (prebuilt for Linux/macOS). Re-running is safe — it skips tools that are already present. Commit the updated binaries afterwards.

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

The Swedish translation (`translation/monkey1/monkey1_swe.txt`) is sourced from the
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

The SE engine limitation means we must replace the French language slot. This is why
users need to select French in the game settings.

---

## Adding support for a new game

### Classic-only games (CD-ROM / ScummVM)

If you have access to the classic game files only (e.g. `MONKEY1.000` / `MONKEY1.001`),
the classic workflow works standalone — no Special Edition required:

1. Find the scummtr game ID for your game (see table below, or run `bin/linux/scummtr -L`).
2. Extract the English strings:
   ```
   bash scripts/classic/extract_text.sh <game_id> /path/to/game/ translation/<game>/text.txt
   ```
3. Translate `translation/<game>/text.txt`.
4. Inject the translation back:
   ```
   bash scripts/classic/inject_text.sh <game_id> /path/to/game_copy/ translation/<game>/text.txt
   ```
5. Add a translation directory under `translation/<game>/` and a new patcher command under `cmd/classic-patcher/` (or extend the existing one) following the Monkey Island 1 pattern.

The distributable end-user patcher (`cmd/classic-patcher/`) patches the game files in-place
and works on Windows, macOS, and Linux — no external tools needed by the end user.

### Special Edition games

1. Investigate the PAK structure with `tools/pak.py extract`.
2. Determine whether the game uses embedded classic SCUMM files (use the scummtr approach)
   or SE-specific `.info` text files (use `tools/text.py` approach).
3. Check `docs/OPEN_QUESTIONS.md` and `docs/FRS.md` for notes on future game support.
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
