#!/usr/bin/env python3
"""Decode SCUMM v5 256-color room background to PNG.

SMAP offset formula (standard SCUMM v5, not GF_SMALL_HEADER):
  strip[n] starts at: smap_block_start + LE_UINT32(smap_block_start + 8 + n*4)

Usage:
    python3 tools/decode_room.py <LFLF_dir> <output.png>
    python3 tools/decode_room.py games/monkey1/gen/full_dump/DISK_0001/LECF/LFLF_0028 room_028.png
"""

import sys
import os
from PIL import Image

from scumm_gfx import be32, le32, le16, find_block, decode_strip


# ---------------------------------------------------------------------------
# Main decoder
# ---------------------------------------------------------------------------

def decode_room(lflf_dir, out_path):
    room_dir = os.path.join(lflf_dir, 'ROOM')

    # --- RMHD: room dimensions ---
    with open(os.path.join(room_dir, 'RMHD'), 'rb') as f:
        rmhd = f.read()
    width  = le16(rmhd, 8)
    height = le16(rmhd, 10)
    print(f"  Room: {width} x {height}")

    # --- CLUT: 256-colour palette ---
    with open(os.path.join(room_dir, 'CLUT'), 'rb') as f:
        clut = f.read()
    pal_raw = clut[8:]
    palette = [(pal_raw[i*3], pal_raw[i*3+1], pal_raw[i*3+2]) for i in range(256)]

    # --- RMIM: compressed image ---
    with open(os.path.join(room_dir, 'RMIM'), 'rb') as f:
        rmim = f.read()

    # Outer RMIM block starts at 0; find IM00 inside it (after 8-byte header)
    im00_pos = find_block(rmim, 8, len(rmim), 'IM00')
    if im00_pos < 0:
        sys.exit("ERROR: IM00 block not found in RMIM")

    im00_size = be32(rmim, im00_pos + 4)
    smap_pos  = find_block(rmim, im00_pos + 8, im00_pos + im00_size, 'SMAP')
    if smap_pos < 0:
        sys.exit("ERROR: SMAP block not found in IM00")

    num_strips = width // 8
    print(f"  Strips: {num_strips}")

    # Codec survey for diagnostics
    codecs = set()
    for s in range(num_strips):
        off = smap_pos + le32(rmim, smap_pos + 8 + s * 4)
        codecs.add(rmim[off])
    print(f"  Codecs used: {sorted(codecs)}")

    # Decode all strips
    pixels = bytearray(width * height)

    for s in range(num_strips):
        off = smap_pos + le32(rmim, smap_pos + 8 + s * 4)
        if s + 1 < num_strips:
            next_off = smap_pos + le32(rmim, smap_pos + 8 + (s+1) * 4)
        else:
            next_off = smap_pos + be32(rmim, smap_pos + 4)

        strip_data = rmim[off:next_off]
        try:
            strip_pix = decode_strip(strip_data, height)
        except Exception as e:
            print(f"  Strip {s} error (codec={strip_data[0]}): {e}", file=sys.stderr)
            strip_pix = [0] * (8 * height)

        sx = s * 8
        for row in range(height):
            for col in range(8):
                buf_idx = row * width + sx + col
                pix_idx = row * 8 + col
                if buf_idx < len(pixels) and pix_idx < len(strip_pix):
                    pixels[buf_idx] = strip_pix[pix_idx]

    img = Image.new('RGB', (width, height))
    img.putdata([palette[p] for p in pixels])
    img.save(out_path)
    print(f"  Saved: {out_path}")


# ---------------------------------------------------------------------------

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <LFLF_dir> <output.png>")
        print(f"  e.g. {sys.argv[0]} games/monkey1/gen/full_dump/DISK_0001/LECF/LFLF_0028 out.png")
        sys.exit(1)

    decode_room(sys.argv[1], sys.argv[2])
