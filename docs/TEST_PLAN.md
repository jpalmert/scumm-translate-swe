# Test Plan — MI1 Swedish Translation Patchers

## Overview

Tests are organized into three levels:

| Level | Location | Requires | Run command |
|-------|----------|----------|-------------|
| Unit | `internal/*/`, `cmd/*/` | Nothing (synthetic data) | `go test ./...` |
| Integration | `*_integration_test.go` (build tag `integration`) | `game/monkey1/Monkey1.pak` | `go test -tags integration ./...` |
| Manual / acceptance | This document | Working patcher binary + game files | Run manually |

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
second (modified content).

#### BAK-003: Missing source → error
Call `Create` on a non-existent file. Assert error.

#### BAK-004: Backup path is `<original>.bak`
Assert the returned path equals `<original> + ".bak"`.

### Classic patcher (`cmd/classic-patcher`)

#### CLASSIC-001: Missing game directory → error
#### CLASSIC-002: Directory missing `MONKEY1.000` → error
#### CLASSIC-003: Directory missing `MONKEY1.001` → error
#### CLASSIC-004: Translation file not found → error
#### CLASSIC-005: Lowercase filenames accepted (`monkey1.000`)
#### CLASSIC-006: Uppercase preferred over lowercase when both exist
#### CLASSIC-007: `findGameFile` returns error when neither name exists
#### CLASSIC-008: `findTranslationFile` returns error for missing explicit path
#### CLASSIC-009: `findTranslationFile` accepts a valid explicit path

### SE patcher (`cmd/se-patcher`)

#### SE-001: Non-existent input PAK → error
#### SE-002: Invalid PAK magic → error
#### SE-003: PAK missing `classic/en/monkey1.000` → error
#### SE-004: PAK missing `classic/en/monkey1.001` → error
#### SE-005: Translation file not found → error
#### SE-006: In-place mode creates a `.bak` file before injection
A synthetic PAK with fake game data is used; scummtr will fail, but the backup must be
created before the injection step is reached.
#### SE-007: Explicit output path → no backup created for input
#### SE-008: `findTranslationFile` returns error for missing explicit path
#### SE-009: `findTranslationFile` accepts a valid explicit path

---

## Integration Tests (`go test -tags integration ./...`)

These tests require `game/monkey1/Monkey1.pak` and skip gracefully if absent.

### Classic package (`internal/classic`)

#### INT-002: Identity translation is idempotent
1. Extract `MONKEY1.000/.001` from the real PAK.
2. Export English strings with scummtr.
3. Inject those same strings back (identity).
4. Run a second round-trip.

Assert: second round-trip output is byte-identical to the first. (scummtr normalizes
internal structures on first inject, but subsequent injects of the same data are stable.)

#### INT-CLASSIC: Real Swedish translation grows `.001`
Run `classic.InjectTranslation` with the real `monkey1_swe.txt`. Assert `MONKEY1.001`
is larger after injection (Swedish text is longer than English).

### SE patcher (`cmd/se-patcher`)

#### INT-SE-001: Full SE pipeline — patched PAK is valid and `.001` grew
Run `runSEPatch` with the real `Monkey1.pak` and explicit output path. Assert:
- Output PAK is readable by `pak.Read`
- Entry count is identical to input
- `classic/en/monkey1.001` is larger in the output

#### INT-SE-002: In-place mode creates backup with correct content
Copy `Monkey1.pak` to a temp dir, run `runSEPatch` with no explicit output. Assert:
- `Monkey1.pak.bak` exists with the same size and content as the original

---

## Manual Acceptance Tests

These require a working game installation and cannot be automated.

### MAN-001: SE patcher — basic usage (GOG)
1. Run `se-patcher-linux /path/to/Monkey1.pak`
2. Confirm backup `Monkey1.pak.bak` created
3. Launch game, set language to French, start new game
4. Assert: dialog appears in Swedish

### MAN-002: SE patcher — Steam version
Same as MAN-001 with Steam version. Assert: patcher accepts `LPAK` magic without error.

### MAN-003: SE patcher — explicit output path
```
se-patcher-linux Monkey1.pak /tmp/Monkey1_sv.pak
```
Assert: output written to specified path, original untouched, no backup created.

### MAN-004: Classic patcher — ScummVM usage
1. Run `classic-patcher-linux /path/to/game/dir`
2. Confirm `MONKEY1.000.bak` and `MONKEY1.001.bak` created
3. Open ScummVM, set language to French, start new game
4. Assert: dialog appears in Swedish

### MAN-005: Custom translation file
Place a modified `monkey1_swe.txt` next to the patcher binary. Run without specifying a
translation path. Assert: patcher uses the file next to the binary.

### MAN-006: Missing input → helpful error
Pass a non-existent path. Assert: human-readable error, no panic.

### MAN-007: Wrong input file → helpful error
Pass a non-PAK file as input to `se-patcher`. Assert: error message mentions the wrong
magic bytes, not a raw panic.

---

## Running all tests

```bash
# Unit tests only (fast, no game files):
go test ./...

# Unit + integration (requires game/monkey1/Monkey1.pak):
go test -tags integration -v ./...
```
