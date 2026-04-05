# Open Questions & Investigation Items

Questions that must be resolved before or during implementation.
Update status as items are investigated.

---

## OQ-1: GOG vs Steam MI1SE file layout
**Status:** RESOLVED (2026-04-04)  
**Priority:** P0

**Finding:** GOG MI1SE uses the same PAK structure as Steam with one difference:
- Magic bytes are `KAPL` (GOG) instead of `LPAK` (Steam) — same bytes, reversed order
- All header fields, file entry format, and data layout are identical (little-endian)
- File is named `Monkey1.pak` in both versions
- No `.info` files — dialog text lives in the embedded `classic/en/MONKEY1.000`

**Fix applied:** `internal/pak/pak.go` accepts both `LPAK` and `KAPL` magic.

---

## OQ-2: scummtr string format ↔ SE text format alignment
**Status:** RESOLVED (2026-04-04)  
**Priority:** P0

**Finding:** The SE PAK embeds `classic/en/MONKEY1.000` + `MONKEY1.001` — the exact same
classic SCUMM data files that scummtr operates on. There are no `.info` text files.
The game dialog text is driven entirely by these embedded classic files.

Comparison result:
- SE embedded classic: 4437 strings (scummtr game ID `monkeycdalt`)
- monkeycd_swe text.swe: 4437 strings
- **Strings match 1:1 in order** — the SE uses the same classic data as the CD version

**Conclusion:** Option (a) — we use scummtr to inject `text.swe` directly into
`classic/en/MONKEY1.000`, then repack the modified file back into the PAK.
No format conversion or reconciliation step needed.

---

## OQ-3: Self-contained patcher technology
**Status:** RESOLVED (2026-04-05)  
**Priority:** P1

**Decision:** Go single binary (option c).

The patcher is implemented in Go (`cmd/`, `internal/`). It embeds the scummtr binary and
the Swedish translation file at compile time using `go:embed`. The resulting binary:
- Has no runtime dependencies (no Python, no external tools)
- Is ~7-9 MB per platform
- Runs on Linux, macOS, and Windows
- Handles both GOG (KAPL magic) and Steam (LPAK magic) PAK files

Build with `bash scripts/build_patcher.sh` — produces:
- `dist/se-patcher-linux`, `dist/se-patcher-darwin`, `dist/se-patcher-windows.exe`
- `dist/classic-patcher-linux`, `dist/classic-patcher-darwin`, `dist/classic-patcher-windows.exe`
- `dist/monkey1.txt`

---

## OQ-4: French slot UX problem
**Status:** KNOWN LIMITATION — mitigation needed  
**Priority:** P1

The SE engine only supports replacing existing language slots. We replace French.
This means users must set their game language to French to see the Swedish translation.

This is confusing and undesirable for end users.

**Mitigations to investigate:**
- Can the patcher automatically set the language to French in a game config file?
- Is there a config file (e.g. `default.cfg`, registry key, `scummvm.ini`) that can be patched?
- Does the SE have any other mechanism for adding a new language?

---

## OQ-5: Savegame compatibility
**Status:** KNOWN ISSUE — document only  
**Priority:** P2

After patching, existing savegames may fail to load due to checksum verification against resource files. This is the same issue as the original monkeycd_swe project.

**To document:** Clear warning in patcher and README that existing saves may break.
**To investigate:** Is there a way to patch savegame checksums as well?

---

## OQ-6: Multi-version support scope
**Status:** UNRESOLVED  
**Priority:** P1

The GOG and Steam versions of MI1SE may have different file checksums even if the content is the same. The patcher needs to handle this.

Questions:
- How many distinct file versions of MI1SE exist (GOG, Steam, disc, regional variants)?
- Should the patcher support all of them, or just the latest GOG/Steam versions?
- Strategy: maintain a list of known-good source checksums; reject unknowns with a helpful message

---

## OQ-7: Classic MI1 distribution format
**Status:** RESOLVED  
**Priority:** P2

The classic version uses `classic-patcher` (Go binary), which embeds scummtr and patches the game files directly. BPS patch distribution (Floating IPS) was considered but dropped — the Go patcher is self-contained and works on all platforms without any extra tools.

---

## OQ-8: Translation workflow design
**Status:** DEFERRED  
**Priority:** P2

Multi-pass translation workflow design is deferred until end-to-end testing with MI1 pre-translated files is complete. Questions to answer after that milestone:

- What constitutes a "review pass"? What does Claude check for?
- How are translator notes / flagged strings tracked?
- How do we handle strings that need context (character speaking, game state)?
- Should there be a human review step at any point?
