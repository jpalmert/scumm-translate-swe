# scumm-translation

A toolkit for creating fan translations of LucasArts SCUMM engine games,
supporting both classic DOS/CD-ROM versions and Special Edition re-releases.

Translations are done with AI assistance (Claude) — no manual text editing required.

## Quick start

```bash
# 1. Install all dependencies
bash scripts/install_deps.sh

# 2. Add a game to translate (see games/monkey1/ as template)

# Classic version workflow:
bash scripts/classic/extract_text.sh monkeycd ~/games/monkey1/ games/monkey1/text/translation.txt
# → translate games/monkey1/text/translation.txt with Claude
bash scripts/classic/inject_text.sh monkeycd ~/games/monkey1_copy/ games/monkey1/text/translation.txt
bash scripts/classic/build_patch.sh ~/games/monkey1_original/ ~/games/monkey1_copy/ games/monkey1/patches/

# Special Edition workflow:
bash scripts/se/extract_for_translation.sh 1 Monkey1.pak games/monkey1/se_translations/
# → translate JSON files in games/monkey1/se_translations/ with Claude
bash scripts/se/build.sh 1 Monkey1.pak games/monkey1/se_translations/ games/monkey1/patches/Monkey1_translated.pak
```

## Supported games

Any game supported by scummtr (run `tools/bin/scummtr -L` for the full list):
Maniac Mansion, Zak McKracken, Indiana Jones 3 & 4, Loom, Monkey Island 1 & 2,
Day of the Tentacle, Sam & Max, Full Throttle, The Dig.

SE support: Monkey Island 1 SE, Monkey Island 2 SE (Steam versions).

## Repository structure

```
games/<game>/
  text/               Extracted + translated text files (classic)
  se_translations/    Extracted + translated JSON files (SE)
  graphics/           Modified sprites, backgrounds, charsets
  references/         TRANSLATE_TABLE, reverse engineering notes
  patches/            Final distributable patch files

scripts/
  install_deps.sh     One-time dependency installer
  classic/            Classic SCUMM workflow scripts
  se/                 Special Edition workflow scripts

tools/
  bin/                Built tool binaries (scummtr, flips, etc.) — gitignored
  mise/               Custom SE tools (pak.py, text.py, font.py)
```

## Dependencies

| Tool | Purpose | Auto-installed |
|------|---------|----------------|
| scummtr | Classic SCUMM text extract/inject | Yes (built from source) |
| flips | Create/apply BPS patches | Yes (built from source) |
| Pillow | Python image processing (SE fonts) | Yes (pip) |
| nutcracker | SCUMM V5-V8 resource editing | Yes (pip) |
| Wine | Run Windows tools if needed | Yes (apt) |
| scummvm-tools | descumm script decompiler | Yes (apt/source) |

## Important notes

- **Never commit original game files** — .gitignore excludes them
- **Always patch fresh files** — re-patching can corrupt graphics
- **SE translations use the French language slot** — set game language to French
  after applying a translated SE pak
- **Savegames may break** after patching due to checksum changes

## Reference: scummtr game IDs for common games

| Game | ID |
|------|----|
| Monkey Island 1 (CD) | monkeycd |
| Monkey Island 1 (EGA) | monkey |
| Monkey Island 2 | monkey2 |
| Indiana Jones 4 | atlantis |
| Day of the Tentacle | tentacle |
| Sam & Max | samnmax |
| Full Throttle | ft |
| The Dig | dig |
