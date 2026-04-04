#!/usr/bin/env python3
"""
Font glyph expander for Monkey Island Special Edition .font files.
Adds new characters (e.g. Swedish Å/Ä/Ö/å/ä/ö) to the game font.

.font format:
  +0x00  4B  game-specific header data
  +0x04  1B  numGlyphs (uint8) — current glyph count in PNG table
  +0x05  ... various header fields
  +0x5A  ... ASCII→PNG-index lookup table
              address of char c: (ord(c) - 0x20) * 2 + 0x5A
              each entry is 1 byte: PNG table index for that character
  +0x4260 ... PNG glyph table (starts at 0x4260 = 16992)
              each entry is 16 bytes (8 × int16):
                +0x00  leftCol    (1-based pixel column of glyph box in PNG)
                +0x02  topRow     (1-based pixel row)
                +0x04  rightCol   (1-based)
                +0x06  bottomRow  (1-based)
                +0x08  indent     (pixels before glyph starts)
                +0x0A  width      (pixel width of glyph)
                +0x0C  kerning    (advance width after glyph)
                +0x0E  (padding, always 0)

  The companion PNG file contains all glyph images in rows.
  New glyphs are appended as new rows at the bottom of the PNG.

Known numGlyphs:
  MI:SE  (game 1): 0x9A = 154
  MI2:SE (game 2): 0x9B = 155

Usage:
  # Add Swedish characters from a glyph strip PNG
  python3 font.py add-glyphs \\
    --font   original.font \\
    --png    original_font.png \\
    --glyphs swedish_glyphs.png \\
    --map    "Å:91,Ä:92,Ö:93,å:123,ä:124,ö:125,é:130" \\
    --out-font  modified.font \\
    --out-png   modified_font.png

  # Inspect a font file
  python3 font.py inspect original.font

The --map argument maps ASCII codes (or characters) to the new glyph
images in the --glyphs PNG, in left-to-right order.
Format: "char_or_code:ascii_code,..." e.g. "Å:91,Ä:92" or "91,92,93"

The --glyphs PNG should contain one row of glyph images on a transparent
or black background. Each glyph is automatically detected by bounding box.
"""

import struct
import sys
import json
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    print("ERROR: Pillow is required. Run: pip install Pillow")
    sys.exit(1)

GLYPH_TABLE_OFFSET = 0x4260
GLYPH_ENTRY_SIZE   = 16
NUM_GLYPHS_OFFSET  = 0x04
ASCII_LOOKUP_BASE  = 0x5A
ASCII_LOOKUP_STEP  = 2   # 2 bytes per entry; only first byte used


def _read_i16(data, offset):
    return struct.unpack_from('<h', data, offset)[0]

def _write_i16(buf, offset, val):
    struct.pack_into('<h', buf, offset, val)

def _read_u8(data, offset):
    return data[offset]


def read_glyph_table(font_data):
    """Return list of glyph dicts from the font file."""
    num_glyphs = _read_u8(font_data, NUM_GLYPHS_OFFSET)
    glyphs = []
    for i in range(num_glyphs):
        base = GLYPH_TABLE_OFFSET + i * GLYPH_ENTRY_SIZE
        glyphs.append({
            'index':     i,
            'left_col':  _read_i16(font_data, base + 0),
            'top_row':   _read_i16(font_data, base + 2),
            'right_col': _read_i16(font_data, base + 4),
            'bot_row':   _read_i16(font_data, base + 6),
            'indent':    _read_i16(font_data, base + 8),
            'width':     _read_i16(font_data, base + 10),
            'kerning':   _read_i16(font_data, base + 12),
        })
    return glyphs


def read_ascii_lookup(font_data):
    """Return dict mapping ascii_code -> png_table_index."""
    lookup = {}
    # Scan printable ASCII range
    for code in range(0x20, 0xFF):
        addr = (code - 0x20) * ASCII_LOOKUP_STEP + ASCII_LOOKUP_BASE
        if addr + 1 > len(font_data):
            break
        idx = _read_u8(font_data, addr)
        if idx != 0:
            lookup[code] = idx
    return lookup


def detect_glyphs_in_strip(glyphs_png_path, background_color=None):
    """
    Auto-detect glyph bounding boxes in a horizontal glyph strip PNG.
    Returns list of (left, top, right, bottom) tuples in pixel coords (0-based).
    background_color: (R,G,B[,A]) to treat as background; None = auto-detect from corners.
    """
    img = Image.open(glyphs_png_path).convert('RGBA')
    pixels = img.load()
    w, h = img.size

    if background_color is None:
        # Assume corners are background
        background_color = pixels[0, 0]

    def is_bg(px):
        # Compare ignoring alpha if background is opaque
        if len(background_color) == 4 and background_color[3] == 0:
            return px[3] == 0  # fully transparent = background
        return px[:3] == background_color[:3]

    # Find columns that contain non-background pixels
    nonempty_cols = [x for x in range(w)
                     if any(not is_bg(pixels[x, y]) for y in range(h))]

    if not nonempty_cols:
        return []

    # Group consecutive non-empty columns into glyph spans
    glyphs = []
    glyph_start = nonempty_cols[0]
    prev = nonempty_cols[0]
    for col in nonempty_cols[1:]:
        if col > prev + 2:  # gap of >2 pixels = new glyph
            glyphs.append(glyph_start)
            glyph_start = col
        prev = col
    glyphs.append(glyph_start)

    # For each glyph start column, find the bounding box
    boxes = []
    for start in glyphs:
        # Find end column
        x = start
        while x < w:
            if not any(not is_bg(pixels[x, y]) for y in range(h)):
                break
            x += 1
        end = x - 1

        # Find top and bottom rows within this column range
        top = h
        bot = 0
        for col in range(start, end + 1):
            for row in range(h):
                if not is_bg(pixels[col, row]):
                    top = min(top, row)
                    bot = max(bot, row)

        boxes.append((start, top, end, bot))

    return boxes


def parse_char_map(map_str):
    """
    Parse --map argument into list of ascii codes in glyph order.
    Accepts: "Å:91,Ä:92,Ö:93" or "91,92,93" or mixed.
    Returns list of int ascii codes.
    """
    codes = []
    for part in map_str.split(','):
        part = part.strip()
        if ':' in part:
            char, code = part.split(':', 1)
            codes.append(int(code))
        elif part.isdigit():
            codes.append(int(part))
        elif len(part) == 1:
            codes.append(ord(part))
        else:
            raise ValueError(f"Can't parse char map entry: {part!r}")
    return codes


def add_glyphs(font_path, font_png_path, glyphs_png_path, char_map_str,
               out_font_path, out_png_path):
    """
    Add new glyphs to a font file and its companion PNG.

    char_map_str: mapping of glyphs (in order, left-to-right in glyphs_png)
                  to ASCII codes. See parse_char_map().
    """
    font_data = bytearray(Path(font_path).read_bytes())
    ascii_codes = parse_char_map(char_map_str)

    # Detect glyphs in the new glyph strip
    boxes = detect_glyphs_in_strip(glyphs_png_path)
    if len(boxes) != len(ascii_codes):
        raise ValueError(
            f"Glyph count mismatch: detected {len(boxes)} glyphs in PNG "
            f"but {len(ascii_codes)} entries in char map"
        )

    # Load existing font PNG
    font_img = Image.open(font_png_path).convert('RGBA')
    glyph_img = Image.open(glyphs_png_path).convert('RGBA')

    font_w, font_h = font_img.size
    glyph_h = max(b - t + 1 for (_, t, _, b) in boxes) if boxes else 0

    # Determine the row where we'll append new glyphs in the font PNG
    # Convention: append a new row at the bottom
    new_row_top = font_h  # 0-based
    new_font_h = font_h + glyph_h + 2  # 2px padding

    # Create expanded font PNG
    new_font_img = Image.new('RGBA', (font_w, new_font_h), (0, 0, 0, 0))
    new_font_img.paste(font_img, (0, 0))

    # Current glyph count
    num_glyphs = _read_u8(font_data, NUM_GLYPHS_OFFSET)
    new_glyph_idx = num_glyphs

    for i, (ascii_code, box) in enumerate(zip(ascii_codes, boxes)):
        src_left, src_top, src_right, src_bot = box
        glyph_w = src_right - src_left + 1
        glyph_h_actual = src_bot - src_top + 1

        # Position in new font PNG (pack left to right, starting at column 0 for simplicity
        # — real fonts pack tightly; here we place each on its own row for clarity)
        dest_left = 0
        dest_top  = new_row_top + i * (glyph_h + 2)
        if dest_top + glyph_h_actual > new_font_h:
            # Expand more if needed
            additional = dest_top + glyph_h_actual + 2 - new_font_h
            expanded = Image.new('RGBA', (font_w, new_font_h + additional), (0, 0, 0, 0))
            expanded.paste(new_font_img, (0, 0))
            new_font_img = expanded
            new_font_h += additional

        # Paste glyph into font image
        glyph_crop = glyph_img.crop((src_left, src_top, src_right + 1, src_bot + 1))
        new_font_img.paste(glyph_crop, (dest_left, dest_top))

        # Write glyph table entry (1-based coordinates per .font spec)
        entry_offset = GLYPH_TABLE_OFFSET + new_glyph_idx * GLYPH_ENTRY_SIZE
        # Extend font_data if needed
        needed = entry_offset + GLYPH_ENTRY_SIZE
        if needed > len(font_data):
            font_data += b'\x00' * (needed - len(font_data))

        _write_i16(font_data, entry_offset + 0,  dest_left)             # leftCol (1-based: dest_left+1-1 = dest_left)
        _write_i16(font_data, entry_offset + 2,  dest_top + 1)          # topRow (1-based)
        _write_i16(font_data, entry_offset + 4,  dest_left + glyph_w)   # rightCol (1-based)
        _write_i16(font_data, entry_offset + 6,  dest_top + glyph_h_actual)  # bottomRow (1-based)
        _write_i16(font_data, entry_offset + 8,  0)                     # indent
        _write_i16(font_data, entry_offset + 10, glyph_w)               # width
        _write_i16(font_data, entry_offset + 12, glyph_w + 1)           # kerning (width + 1px spacing)
        _write_i16(font_data, entry_offset + 14, 0)                     # padding

        # Update ASCII lookup table
        lookup_addr = (ascii_code - 0x20) * ASCII_LOOKUP_STEP + ASCII_LOOKUP_BASE
        if lookup_addr < len(font_data):
            font_data[lookup_addr] = new_glyph_idx
        else:
            print(f"  WARNING: ASCII code {ascii_code} outside lookup table range")

        print(f"  added glyph for ASCII {ascii_code} ({chr(ascii_code) if ascii_code < 128 else hex(ascii_code)}) "
              f"at table index {new_glyph_idx}, size {glyph_w}×{glyph_h_actual}")

        new_glyph_idx += 1

    # Update numGlyphs
    font_data[NUM_GLYPHS_OFFSET] = new_glyph_idx & 0xFF

    Path(out_font_path).write_bytes(font_data)
    new_font_img.save(out_png_path)
    print(f"\nWrote {out_font_path} (numGlyphs: {num_glyphs} -> {new_glyph_idx})")
    print(f"Wrote {out_png_path}")


def inspect(font_path):
    """Print summary of a font file."""
    data = Path(font_path).read_bytes()
    num_glyphs = _read_u8(data, NUM_GLYPHS_OFFSET)
    lookup = read_ascii_lookup(data)

    print(f"Font file: {font_path}")
    print(f"  numGlyphs: {num_glyphs} (0x{num_glyphs:02X})")
    print(f"\nASCII→glyph mappings ({len(lookup)} entries):")
    for code in sorted(lookup.keys()):
        char_repr = repr(chr(code)) if 0x20 <= code < 0x7F else f"0x{code:02X}"
        print(f"  {code:3d} {char_repr:6s} -> glyph #{lookup[code]}")

    print(f"\nGlyph table (first 10):")
    glyphs = read_glyph_table(data)
    for g in glyphs[:10]:
        print(f"  [{g['index']:3d}] left={g['left_col']} top={g['top_row']} "
              f"right={g['right_col']} bot={g['bot_row']} "
              f"width={g['width']} kerning={g['kerning']}")
    if len(glyphs) > 10:
        print(f"  ... ({len(glyphs) - 10} more)")


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    cmd = sys.argv[1]

    if cmd == 'inspect' and len(sys.argv) >= 3:
        inspect(sys.argv[2])

    elif cmd == 'add-glyphs':
        import argparse
        parser = argparse.ArgumentParser(prog='font.py add-glyphs')
        parser.add_argument('--font',      required=True)
        parser.add_argument('--png',       required=True)
        parser.add_argument('--glyphs',    required=True)
        parser.add_argument('--map',       required=True)
        parser.add_argument('--out-font',  required=True)
        parser.add_argument('--out-png',   required=True)
        args = parser.parse_args(sys.argv[2:])
        add_glyphs(args.font, args.png, args.glyphs, args.map,
                   args.out_font, args.out_png)
    else:
        print(__doc__)
        sys.exit(1)


if __name__ == '__main__':
    main()
