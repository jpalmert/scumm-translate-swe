# Test Plan — MI1 Swedish Translation Patchers

## Overview

Tests are organized into several levels. The unified test script `scripts/test.sh`
runs them all:

```bash
bash scripts/test.sh monkey1        # unit + Python (no game files needed)
bash scripts/test.sh monkey1 --all  # also buildpatcher + integration (needs game files)
```

Without `--all`, only tests that need no game files or build artifacts run.
With `--all`, buildpatcher and integration tests also run — missing prerequisites
cause a **FAIL**, not a skip.

| Level | Location | Requires | `test.sh` | Individual command |
|-------|----------|----------|-----------|--------------------|
| Unit | `internal/*/`, `cmd/*/` | Nothing (synthetic data) | default | `go test ./...` |
| Python | `tools/test_*.py` | Python 3 | default | `python3 -m unittest discover -s tools -p 'test_*.py'` |
| Build-patcher | `internal/charset/` (build tag `buildpatcher`) | `scripts/build.sh` step 2 | `--all` | `go test -tags buildpatcher ./internal/charset/...` |
| Integration | `*_integration_test.go` (build tag `integration`) | `games/monkey1/game/Monkey1.pak` | `--all` | `go test -tags integration ./...` |
| Manual / acceptance | This document | Working patcher binary + game files | n/a | Run manually |

---

## Unit Tests (`go test ./...`)

These tests use synthetic in-memory data — no game files required.

### PAK package (`internal/pak`)

#### PAK-001: Round-trip (Steam magic LPAK)
Build a synthetic PAK with 3 files using `LPAK` magic, write to temp file, read back, write
again. Assert the second write is byte-identical to the first and magic is preserved.

#### PAK-002: Round-trip (GOG magic KAPL)
Same as PAK-001 with `KAPL` magic. Specifically asserts the output starts with `KAPL`,
not silently rewritten to `LPAK`.

#### PAK-003: DataPos recalculation after size change
Read a synthetic PAK, replace one entry with larger data, write back, re-read. Assert all
`DataPos` values are contiguous (no gaps or overlaps) and the other entries are unchanged.

#### PAK-004: Invalid magic → error
File with wrong magic (`XPAK`). Assert `pak.Read` returns an error.

#### PAK-005: File too small → error
10-byte file. Assert `pak.Read` returns an error.

#### PAK-006: Classic file entries found by name
PAK containing `classic/en/monkey1.000` and `classic/en/monkey1.001`. Assert both entries
are present in the result.

### Backup package (`internal/backup`)

#### BAK-001: Create backup
Copy a file to `path.bak`. Assert backup exists with same content.

#### BAK-002: Create does not overwrite existing backup
Call `Create` twice. Assert the backup reflects the first call (original content), not the
second (modified content). Returns `ErrBackupExists`.

#### BAK-003: Missing source → error
Call `Create` on a non-existent file. Assert error.

#### BAK-004: Backup path is `<original>.bak`
Assert the returned path equals `<original> + ".bak"` for MONKEY1.000, MONKEY1.001, and
Monkey1.pak.

### Font package (`internal/font`)

#### FONT-001: Swedish characters remapped to correct glyphs
Populate Windows-1252 glyph entries (0xC5=Å, 0xC4=Ä, etc.), run `RemapLookup`.
Assert each SCUMM internal code (91, 92, 93, 123, 124, 125, 130) maps to the same
glyph as its corresponding Windows-1252 code.

#### FONT-002: Input data is not modified (returns a copy)
Assert the original slice is byte-identical before and after `RemapLookup`.

#### FONT-003: Error when source unicode code has no glyph (index 0)
Assert error when a required Windows-1252 glyph position is zero (unmapped).

#### FONT-004: Error when font data is too small for a lookup address
Assert error when font buffer is smaller than the highest required lookup address.

#### FONT-005: Existing glyph mappings for unrelated characters are preserved
Assert that entries not in `SwedishRemapping` are unchanged after remap.

#### FONT-006: Applying the same remapping twice is idempotent
Assert the second `RemapLookup` on already-remapped data produces the same output.

#### FONT-007: Error when destination SCUMM code address is out of range
Assert error when the destination address for a remapped SCUMM code exceeds the font
data buffer size.

### Classic package — encoding (`internal/classic`)

#### ENCODE-001: Swedish UTF-8 characters → SCUMM escape codes
Each Swedish character (Å, Ä, Ö, å, ä, ö, é) is written as UTF-8 to a temp file and
passed through `encodeForScummtr`. Assert the output contains the correct `\NNN` escape
code for each character.

#### ENCODE-002: ASCII bytes passed through unchanged
Plain ASCII input (`Hello, world!\n`). Assert output is identical to input.

#### ENCODE-003: Mixed content — Swedish chars encoded, rest unchanged
Input `Jag är glad`. Assert `ä` is encoded as `\124` while ASCII characters pass through.

#### ENCODE-004: Empty-content lines get a space injected
Lines with a header but no text content (e.g. `[002:SCRP#0037]`) get a single space
injected so scummtr accepts them while preserving sequential matching.

#### ENCODE-005: Whitespace-only content preserved as-is
Lines that already contain a single space are not double-padded.

#### ENCODE-006: Opcode prefixes stripped before injection
Lines with `(__)` or `(D8)` opcode prefixes have the prefix removed. Swedish characters
in the remaining text are still encoded.

#### ENCODE-007: `DecodeScummtrEscapes` converts `\NNN` escapes to raw bytes
Assert that backslash-escaped decimal byte values in scummtr output are decoded back to
their raw byte values.

### SE patcher (`cmd/patcher`)

#### SE-001: Non-existent input PAK → error
#### SE-002: Invalid PAK magic → error
#### SE-003: PAK missing `classic/en/monkey1.000` → error
#### SE-004: PAK missing `classic/en/monkey1.001` → error
#### SE-005: Translation file not found → error
#### SE-006: In-place mode creates a `.bak` file before injection
A synthetic PAK with fake game data is used; scummtr will fail, but the backup must be
created before the injection step is reached.
#### SE-007: Explicit output path → no backup created for input
#### SE-010: `remapFontEntries` patches `.font` entries and skips others
Synthetic font data with Swedish glyphs at Windows-1252 positions. Assert all 8 SCUMM codes
(91=Å, 92=Ä, 93=Ö, 123=å, 124=ä, 125=ö, 130=é, 136=ê) map to the expected glyph indices. Assert non-font entries unchanged.
#### SE-011: `remapFontEntries` returns error when a font is missing a required glyph
#### SE-012: `remapFontEntries` with no `.font` entries returns 0, nil (graceful no-op)

### Classic patcher (`cmd/patcher`)

#### CLASSIC-001: Missing game directory → error
#### CLASSIC-002: Directory missing `MONKEY1.000` → error
#### CLASSIC-003: Directory missing `MONKEY1.001` → error
#### CLASSIC-004: Translation file not found → error
#### CLASSIC-005: Backup content matches original for both game files
Inject into a dir with fake game data. Assert `.bak` files contain the original bytes.
#### CLASSIC-005c: Lowercase filenames accepted (`monkey1.000`)
#### CLASSIC-006: Uppercase preferred over lowercase when both exist
#### CLASSIC-007: `findGameFile` returns error when neither name exists
#### CLASSIC-008: `findGameFile` accepts alternate naming (`MONKEY.000` without "1")

### Shared helpers (`cmd/patcher`)

#### SHARED-001: `findTranslationFile` returns error for missing explicit path
#### SHARED-002: `findTranslationFile` accepts a valid explicit path

### Auto-detection (`cmd/patcher`)

#### DETECT-001: `isSEInput` returns true for a `.pak` file
#### DETECT-002: `isSEInput` returns false for a directory
#### DETECT-003: `isSEInput` returns true for a non-existent `.pak` path (by extension)

### List PAK (`cmd/patcher`)

#### LIST-001: `runListPAK` lists PAK entries to stdout
Run `runListPAK` on a synthetic PAK and verify it prints entry names.

### Charset package — verb layout (`internal/charset`)

#### VERB-001: `patchVerbCoords` patches verb button coordinates for Swedish labels
#### VERB-002: `findVerbXOffset` finds the correct X-offset for verb buttons
#### VERB-003: `findFileInTree` locates files in nested directory structures

---

## Build-Patcher Tests (`go test -tags buildpatcher ./internal/charset/...`)

These tests validate the embedded CHAR assets and only apply after `scripts/build.sh`
step 2 has generated the `.bin` files.

### Charset package (`internal/charset`)

#### ASSET-001..005: Embedded CHAR assets are valid CHAR blocks
Each of the 5 patched CHAR blocks (0001, 0002, 0003, 0004, 0006) is checked: at least
8 bytes long, starts with `CHAR` tag, and the big-endian size field matches the actual
data length.

#### ASSET-007: Embedded scummrp binaries are non-empty
Assert that the embedded scummrp binaries for Linux, macOS, and Windows are all non-empty.

---

## Integration Tests (`go test -tags integration ./...`)

These tests require `games/monkey1/game/Monkey1.pak` and skip gracefully if absent.

### Classic package (`internal/classic`)

#### INT-002: Identity translation is idempotent
1. Extract `MONKEY1.000/.001` from the real PAK.
2. Export English strings with scummtr.
3. Inject those same strings back (identity).
4. Run a second round-trip.

Assert: second round-trip output is byte-identical to the first. (scummtr normalizes
internal structures on first inject, but subsequent injects of the same data are stable.)

#### INT-CLASSIC: Real Swedish translation grows `.001`
Run `classic.InjectTranslation` with the real `swedish.txt`. Assert `MONKEY1.001`
is larger after injection (Swedish text is longer than English).

#### INT-EXTRACT-PAK: Extracting strings from PAK-sourced classic files
Extract `MONKEY1.000/.001` from the real PAK, write to a temp dir with uppercase names,
run scummtr export. Assert output is non-empty. Mirrors the PAK input mode of
`scripts/extract_pak.sh + scripts/extract_assets.sh`.

#### INT-EXTRACT-DIR: Extracting strings from a classic files directory (uppercase and lowercase)
Two subtests — write classic files as UPPERCASE and as lowercase, copy to work dir
with normalised uppercase names, run scummtr export. Assert output is non-empty.
Mirrors the directory input mode of `scripts/extract_pak.sh + scripts/extract_assets.sh`.

#### INT-ROUNDTRIP: InjectTranslation round-trip with English text is idempotent
Export original English strings in InjectTranslation-compatible format, inject them
back using the production pipeline, re-export, and compare. Assert text is identical.
A second inject+export cycle must also match (idempotence). Catches bugs in our
flag choices, `encodeForScummtr` pre-processing, or temp-file handling.

### SE patcher (`cmd/patcher`)

#### INT-SE-001: Full SE pipeline — patched PAK is valid, `.001` grew, fonts patched
Run `runSEPatch` with the real `Monkey1.pak` and explicit output path. Assert:
- Output PAK is readable by `pak.Read`
- Entry count is identical to input
- `classic/en/monkey1.001` is larger in the output (Swedish text is longer)
- At least one `.font` entry has SCUMM code 91 (Å) remapped to a non-zero glyph

#### INT-SE-002: In-place mode creates backup with correct content
Copy `Monkey1.pak` to a temp dir, run `runSEPatch` with no explicit output. Assert:
- `Monkey1.pak.bak` exists with the same size and content as the original

#### INT-SE-003: Re-patch after manual backup restore succeeds
Patch a copy of `Monkey1.pak` in-place, restore from the `.bak` file, then patch again.
Assert the second patch succeeds without errors (no "CHAR block not found" or similar).

#### INT-SE-004: Automatic re-patch without manual restore
Patch a copy twice without manual restore. Assert the second patch reads from the
backup automatically and both patches produce byte-identical output.

### Classic patcher (`cmd/patcher`)

#### INT-CLASSIC-001: Real Swedish translation grows `.001`
Extract classic files from the PAK, run `runClassicPatch` with the real `swedish.txt`.
Assert `MONKEY1.001` is larger after patching.

#### INT-CLASSIC-002: Classic in-place backup has correct content
Run `runClassicPatch` on extracted classic files. Assert `.bak` files exist for both
`MONKEY1.000` and `MONKEY1.001` with content identical to the originals.

#### INT-CLASSIC-003: Classic re-patch succeeds and is idempotent
Patch classic files twice without manual restore. Assert the second patch succeeds
and produces output identical to the first patch.

---

## Python Tool Tests (`python -m pytest tools/`)

### `test_calc_padding.py`
Tests for `tools/calc_padding.py` — `@` padding logic for SE name buffers.

### `test_scumm_gfx.py`
Tests for `tools/decode_room.py` and `tools/decode_object.py` — SCUMM v5 graphics decoding.

### `test_find_dynamic_names.py`
Tests for `tools/find_dynamic_names.py` — runtime name-change mapping extraction.

### `test_pak.py`
Tests for `tools/pak.py` — PAK archive extraction and repacking.

---

## Manual Acceptance Tests

These require a working game installation and cannot be automated.

### MAN-001: SE patcher — basic usage (GOG)
1. Run `mi1-translate-linux /path/to/Monkey1.pak`
2. Confirm backup `Monkey1.pak.bak` created
3. Launch game, start new game
4. Assert: dialog appears in Swedish with correct characters (å, ä, ö, Å, Ä, Ö, é)

### MAN-002: SE patcher — Steam version
Same as MAN-001 with Steam version. Assert: patcher accepts `LPAK` magic without error.

### MAN-003: SE patcher — explicit output path
```
mi1-translate-linux Monkey1.pak /tmp/Monkey1_sv.pak
```
Assert: output written to specified path, original untouched, no backup created.

### MAN-004: Classic patcher — ScummVM usage
1. Run `mi1-translate-linux /path/to/game/dir`
2. Confirm `MONKEY1.000.bak` and `MONKEY1.001.bak` created
3. Open ScummVM, start new game
4. Assert: dialog appears in Swedish with correct characters

### MAN-005: Custom translation file
Place a modified `swedish.txt` next to the patcher binary. Run without specifying a
translation path. Assert: patcher uses the file next to the binary.

### MAN-006: Missing input → helpful error
Pass a non-existent path. Assert: human-readable error, no panic.

### MAN-007: Wrong input file → helpful error
Pass a non-PAK file as input to `mi1-translate`. Assert: error message mentions the wrong
magic bytes, not a raw panic.

### MAN-008: Swedish characters render correctly in SE
Launch the patched SE game, start a new game. Navigate to a scene with å, ä, ö, Å, Ä, Ö.
Assert: characters render as Swedish letters, not squares or wrong punctuation.
(This is the critical end-to-end test for the font lookup table patching.)

---

## Running all tests

```bash
# From the game directory (no argument needed):
cd games/monkey1 && bash ../../scripts/test.sh

# Or from the repo root with explicit game name:
bash scripts/test.sh monkey1

# Include game-file tests (buildpatcher + integration):
bash scripts/test.sh monkey1 --all
```

Without `--all`, runs Go unit tests and Python tests — no game files or build
artifacts required. With `--all`, also runs buildpatcher asset tests and
integration tests. Missing prerequisites (`.bin` files, `Monkey1.pak`) cause
a **FAIL**, not a skip.

Individual suites can also be run directly if needed:

```bash
go test ./...                                        # Go unit tests
go test -tags buildpatcher ./internal/charset/...    # charset asset tests (after build.sh)
go test -tags integration ./...                      # Go integration tests
python3 -m unittest discover -s tools -p 'test_*.py' # Python tool tests
```
