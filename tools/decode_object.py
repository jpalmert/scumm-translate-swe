#!/usr/bin/env python3
"""
Decode SCUMM v5 object images (OBIM) to PNG.
Usage: python3 decode_object.py <obim_file> <output.png>
"""

import os
import struct
import sys
from PIL import Image

from scumm_gfx import be32, le32, le16, find_block, decode_strip


def decode_object(obim_path, out_path, room_dir=None):
    """Decode OBIM file to PNG

    Args:
        obim_path: Path to OBIM file
        out_path: Output PNG path
        room_dir: Optional path to room directory containing CLUT for palette
    """
    with open(obim_path, 'rb') as f:
        obim = f.read()

    # OBIM structure: OBIM tag, then IMHD (header), IMAG (image data)
    if obim[:4] != b'OBIM':
        raise ValueError("Not an OBIM file")

    # Find IMHD (image header)
    imhd_pos = find_block(obim, 8, len(obim), 'IMHD')
    if imhd_pos < 0:
        raise ValueError("IMHD block not found")

    # IMHD structure (v5), all after 8-byte block header:
    #   +0  obj_id  (u16)
    #   +2  num_imnn (u16)
    #   +4  num_zpnn (u16)
    #   +6  flags   (u8)
    #   +7  unk     (u8)
    #   +8  x       (s16)
    #   +10 y       (s16)
    #   +12 width   (u16)
    #   +14 height  (u16)
    width  = le16(obim, imhd_pos + 20)
    height = le16(obim, imhd_pos + 22)
    num_strips = (width + 7) // 8

    # Load palette from room CLUT if available
    palette = None
    if room_dir:
        clut_path = os.path.join(room_dir, 'CLUT')
        if os.path.exists(clut_path):
            with open(clut_path, 'rb') as f:
                clut = f.read()
            pal_raw = clut[8:]  # Skip CLUT tag + size
            palette = [(pal_raw[i*3], pal_raw[i*3+1], pal_raw[i*3+2]) for i in range(256)]

    # Find image data - usually in IM01
    im01_pos = find_block(obim, imhd_pos, len(obim), 'IM01')
    if im01_pos < 0:
        # Object has no image data (empty/placeholder object)
        return

    im01_size = be32(obim, im01_pos + 4)
    smap_pos = find_block(obim, im01_pos + 8, im01_pos + im01_size, 'SMAP')
    if smap_pos < 0:
        raise ValueError("SMAP not found in IM01 block")

    smap_size = be32(obim, smap_pos + 4)

    # Decode strips
    pixels = bytearray(width * height)
    codecs_used = set()

    for s in range(num_strips):
        strip_off = smap_pos + le32(obim, smap_pos + 8 + s * 4)
        if s + 1 < num_strips:
            next_off = smap_pos + le32(obim, smap_pos + 8 + (s+1) * 4)
        else:
            next_off = smap_pos + smap_size

        strip_data = obim[strip_off:next_off]
        if len(strip_data) == 0:
            continue
        codec = strip_data[0]
        codecs_used.add(codec)

        try:
            strip_pixels = decode_strip(strip_data, height)
            sx = s * 8
            for row in range(height):
                for col in range(8):
                    x = sx + col
                    if x < width:
                        pixels[row * width + x] = strip_pixels[row * 8 + col]
        except Exception as e:
            print(f"  Strip {s} error (codec={codec}): {e}", file=sys.stderr)

    # Create image with palette or grayscale
    try:
        if palette:
            # True-color RGB image
            img = Image.new('RGB', (width, height))
            rgb_pixels = [palette[p] for p in pixels]
            img.putdata(rgb_pixels)
        else:
            # Grayscale (palette indices)
            img = Image.new('L', (width, height))
            img.putdata(list(pixels))

        img.save(out_path)
    except Exception as e:
        print(f"  Error saving image: {e}", file=sys.stderr)
        return

    print(f"  Object: {width} x {height}")
    print(f"  Strips: {num_strips}")
    print(f"  Codecs used: {sorted(codecs_used)}")
    print(f"  Saved: {out_path}")

if __name__ == '__main__':
    if len(sys.argv) < 3 or len(sys.argv) > 4:
        print(f"Usage: {sys.argv[0]} <obim_file> <output.png> [room_dir]")
        print(f"  room_dir: Optional path to room directory with CLUT for palette")
        sys.exit(1)

    room_dir = sys.argv[3] if len(sys.argv) == 4 else None
    decode_object(sys.argv[1], sys.argv[2], room_dir)
