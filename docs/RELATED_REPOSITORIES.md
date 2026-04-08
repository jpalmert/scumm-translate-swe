# Related Repositories and Resources

## Primary References

### 1. monkeycd_swe — Swedish Translation (CD-ROM version)
**URL**: https://github.com/thanius/monkeycd_swe  
**Author**: Tobias Hellgren (hellgren.tobias@gmail.com)  
**Status**: Complete (v1.0, 2021-11-13)

Complete Swedish fan translation of *The Secret of Monkey Island* CD-ROM version.
- **48 PNG graphics files** with Swedish text (signs, title cards, animations)
- **~4,400 lines** of translated dialogue in scummtr format
- **5 custom font files** with Swedish diacriticals (Å, Ä, Ö, å, ä, ö, é)
- **BPS patches** for MONKEY.000 / MONKEY.001

**Key Content**:
- `src/TEXT/text.swe` — Full Swedish dialogue corpus
- `src/GRAPHICS/IMAGES/` — Translated graphics (room backgrounds, objects, costumes)
- `src/GRAPHICS/CHARSETS/` — Custom fonts with Swedish characters
- `src/REFERENCES/TRANSLATE_TABLE` — SCUMM character code mappings
- `patches/` — BPS binary patches for distribution

**Workflow**:
```bash
iconv -f iso-8859-1 -t utf-8 text.swe -o text
sed 's/Å/\\091/g;s/Ä/\\092/g;...' text  # Encode Swedish chars
wine scummtr.exe -gp monkeycd MONKEY/ -i text
```

---

### 2. scummtr — SCUMM Text Resource Tool
**URL**: https://github.com/dwatteau/scummtr  
**Maintainer**: dwatteau (current fork)  
**Original**: Jörg Walter

Command-line tool for extracting and injecting text from SCUMM engine games.

**Capabilities**:
- Extract all dialogue, object names, room descriptions to text format
- Inject modified text back into game resource files (MONKEY.000/001)
- Supports SCUMM v3-v8 games
- Text format: `[room:TYPE#resnum]text content`

**Usage**:
```bash
# Extract
scummtr -p <game_dir> -g <game_id> -ot <output.txt>

# Inject
scummtr -p <game_dir> -g <game_id> -i <input.txt>
```

**Game IDs for MI1**:
- `monkey` — EGA version
- `monkeyega` — EGA talkie
- `monkeyvga` — VGA floppy
- `monkeycd` — CD-ROM version (monkeycd_swe target)
- `monkeycdalt` — CD-ROM alternate

---

### 3. ScummVM Tools — scummrp and others
**URL**: https://github.com/scummvm/scummvm-tools  
**Project**: ScummVM Team

Collection of tools for ScummVM game data manipulation.

**Key Tools**:
- **scummrp** — Extract raw SCUMM resource blocks (RMIM, OBIM, CLUT, etc.)  
  Does NOT decode images, only extracts binary blocks
- **descumm** — SCUMM script disassembler
- **extract_scumm_mac** — Decompress Mac SCUMM archives

**scummrp Usage**:
```bash
# Extract all blocks from a room
scummrp -g <game_id> -p <game_dir> -od <output_dir>

# Extract specific block type
scummrp -g <game_id> -p <game_dir> -t RMIM -od <output_dir>
```

Outputs raw binary blocks that can be decoded with custom tools (like our `decode_room.py` and `decode_object.py`).

---

### 4. MISETranslator — MI1/MI2 Special Edition Tool
**URL**: https://github.com/ShadowNate/MISETranslator  
**Author**: ShadowNate

GUI tool for translating Monkey Island Special Edition (Steam version).

**Features**:
- Extracts `.info` text files from SE PAK archives
- Displays English and Translation columns side-by-side
- Saves as JSON with metadata
- **Targets Steam version**, not GOG

**Relevance**:
- Reference for SE file formats (PAK, .info, .font)
- Our `tools/mise/` Python tools are derived from this project's format specs
- **GOG compatibility unknown** — OQ-1 blocker

---

### 5. ScummVM — Engine Source Code
**URL**: https://github.com/scummvm/scummvm  
**Project**: ScummVM Team  
**License**: GPL v2+

Reference implementation for SCUMM v5 graphics decoders.

**Key Source Files**:
- `engines/scumm/gfx.cpp` — Room/object image decoders
  - `drawStripBasicH()` / `drawStripBasicV()` — ZIGZAG codecs
  - `drawStripComplex()` — MAJMIN codec
  - `FILL_BITS` / `READ_BIT` macros
- `engines/scumm/gfx.h` — Codec enum definitions (BMCOMP_*)
- `engines/scumm/resource.cpp` — Resource loading

Our `tools/decode_room.py` and `tools/decode_object.py` are Python reimplementations of these C++ decoders.

---

## Game Versions and Files

### Classic SCUMM (v5)
**Target for monkeycd_swe**: CD-ROM version  
**Files**: `MONKEY.000` (index) + `MONKEY.001` (data)  
**Size**: ~20-30 MB combined

**File Structure**:
```
MONKEY.000/001 (SCUMM resource bundle)
├── DISK_0001/LECF/
│   ├── LFLF_0001/ (room 001)
│   │   └── ROOM/
│   │       ├── RMHD (room header: width, height)
│   │       ├── CLUT (palette: 256×RGB)
│   │       ├── RMIM (room image: background)
│   │       │   └── IM00/SMAP (strip data)
│   │       ├── OBIM_NNNN (object images)
│   │       ├── OBCD_NNNN (object code)
│   │       ├── LSCR_NNNN (local scripts)
│   │       └── ...
│   ├── LFLF_0002/ (room 002)
│   └── ...
└── Global Resources (CHAR, COST, SOUN, etc.)
```

### Special Edition
**Platforms**: Steam (confirmed), GOG (unknown — OQ-1)  
**Files**: `Monkey1.pak` (main archive) + individual .info/.font files  
**Size**: ~3 GB (with HD graphics/audio)

**File Structure**:
```
Monkey1.pak (archive)
├── classic/ (embedded SCUMM v5 files)
│   ├── en/ (English MONKEY.000/001)
│   └── (other languages)
├── graphics/ (HD backgrounds, objects)
├── audio/ (voice acting, music)
└── ...

Separate files:
├── *.info (text resources: dialogue, UI)
└── *.font (bitmap fonts for text rendering)
```

**SE Format Details**: See `tools/mise/README.md`

---

## This Project's Custom Tools

### tools/decode_room.py
Decodes SCUMM v5 room backgrounds (RMIM → SMAP strips) to PNG.
- Implements all v5 codecs: RAW256 (1), ZIGZAG_V/H (14-48), MAJMIN (64-128)
- Reads palette from CLUT block, outputs true-color PNG

### tools/decode_object.py  
Decodes SCUMM v5 object images (OBIM → IM01 → SMAP strips) to PNG.
- Same codec support as room decoder
- Currently outputs grayscale (palette indices) — needs room palette for colors

### tools/mise/pak.py
Extracts/repacks PAK archives (MI1SE/MI2SE).

### tools/mise/text.py
Extracts/injects text from `.info` files (all SE formats).

### tools/mise/font.py
Expands `.font` glyph tables for Swedish characters.

---

## Open Questions

**OQ-1**: Does GOG MI1SE use the same file structure as Steam?  
Need to verify PAK layout, .info format, and file locations.

**OQ-2**: Do scummtr string IDs align with SE .info strings?  
Can we use monkeycd_swe's `text.swe` to populate SE translations directly?

See `docs/OPEN_QUESTIONS.md` for all 8 open questions.

---

## Useful References

- **SCUMM File Format Wiki**: http://wiki.scummvm.org/index.php/SCUMM/Technical_Reference
- **ScummVM Doxygen** (C++ API docs): https://doxygen.scummvm.org/
- **LucasArts SCUMM Archive**: https://scumm.fandom.com/wiki/Category:SCUMM
