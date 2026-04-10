#!/usr/bin/env python3
"""Decode SCUMM v5 256-color room background to PNG.

Codec reference (from ScummVM engines/scumm/gfx.h / gfx.cpp):
  1       = BMCOMP_RAW256           raw pixels
  14-18   = BMCOMP_ZIGZAG_V4..V8   BasicV,  _decomp_shr = code%10
  24-28   = BMCOMP_ZIGZAG_H4..H8   BasicH,  _decomp_shr = code%10
  34-38   = BMCOMP_ZIGZAG_VT4..VT8 BasicV + transparent
  44-48   = BMCOMP_ZIGZAG_HT4..HT8 BasicH + transparent
  64-68   = BMCOMP_MAJMIN_H4..H8   MajMinCodec horizontal
  84-88   = BMCOMP_MAJMIN_HT4..HT8 MajMinCodec horizontal + transparent
  104-108 = BMCOMP_RMAJMIN_H4..H8  MajMinCodec horizontal (run variant)
  124-128 = BMCOMP_RMAJMIN_HT4..HT8

SMAP offset formula (standard SCUMM v5, not GF_SMALL_HEADER):
  strip[n] starts at: smap_block_start + LE_UINT32(smap_block_start + 8 + n*4)

Usage:
    python3 tools/decode_room.py <LFLF_dir> <output.png>
    python3 tools/decode_room.py game/monkey1/gen/full_dump/DISK_0001/LECF/LFLF_0028 room_028.png
"""

import struct
import sys
import os
from PIL import Image


# ---------------------------------------------------------------------------
# Block parsing helpers
# ---------------------------------------------------------------------------

def be32(data, pos):
    return struct.unpack_from('>I', data, pos)[0]

def le32(data, pos):
    return struct.unpack_from('<I', data, pos)[0]

def le16(data, pos):
    return struct.unpack_from('<H', data, pos)[0]

def find_block(data, start, end, tag):
    """Return byte offset of the first block with this 4-char tag, or -1."""
    pos = start
    tag_b = tag.encode('ascii')
    while pos + 8 <= end:
        if data[pos:pos+4] == tag_b:
            return pos
        size = be32(data, pos+4)
        if size < 8:
            break
        pos += size
    return -1


# ---------------------------------------------------------------------------
# Strip decoders (translated from ScummVM C++)
# ---------------------------------------------------------------------------

def decode_strip_basic(src, height, decomp_shr, decomp_mask, horizontal):
    """
    Gdi::drawStripBasicH (horizontal=True) and drawStripBasicV (horizontal=False).

    Macros from ScummVM gfx.cpp (at offset ~120054):
      READ_BIT  = (cl--, bit = bits & 1, bits >>= 1, bit)
      FILL_BITS = if (cl <= 8): bits |= (*src++ << cl); cl += 8
    """
    pos   = 0
    color = src[pos]; pos += 1
    bits  = src[pos]; pos += 1
    cl    = 8
    inc   = -1

    pixels = [0] * (8 * height)

    def fill_bits():
        nonlocal bits, cl, pos
        if cl <= 8:                         # ← ScummVM: cl <= 8
            if pos < len(src):
                bits |= src[pos] << cl
                pos += 1
            cl += 8

    def read_bit():
        nonlocal bits, cl
        cl -= 1
        b = bits & 1
        bits >>= 1
        return b

    if horizontal:
        # drawStripBasicH: row-by-row
        for row in range(height):
            for col in range(8):
                fill_bits()
                pixels[row * 8 + col] = color
                if not read_bit():
                    pass  # no change
                elif not read_bit():
                    fill_bits()
                    color = bits & decomp_mask
                    bits >>= decomp_shr
                    cl -= decomp_shr
                    inc = -1
                elif not read_bit():
                    color = (color + inc) & 0xff
                else:
                    inc = -inc
                    color = (color + inc) & 0xff
    else:
        # drawStripBasicV: column-by-column
        for col in range(8):
            for row in range(height):
                fill_bits()
                pixels[row * 8 + col] = color
                if not read_bit():
                    pass  # no change
                elif not read_bit():
                    fill_bits()
                    color = bits & decomp_mask
                    bits >>= decomp_shr
                    cl -= decomp_shr
                    inc = -1
                elif not read_bit():
                    color = (color + inc) & 0xff
                else:
                    inc = -inc
                    color = (color + inc) & 0xff

    return pixels


def decode_strip_majmin(src, height, decomp_shr):
    """
    MajMinCodec::decodeLine — used for BMCOMP_MAJMIN_H* and RMAJMIN_H*.

    setupBitReader: color = src[0]; bits = src[1] | src[2]<<8; numBits=16; dataPtr=src+3
    FILL_BITS: if numBits <= 8: bits |= (*dataPtr++) << numBits; numBits += 8
    readBits(n): fill; value = bits & ((1<<n)-1); bits >>= n; numBits -= n
    decodeLine for 8 pixels per row, height rows.
    """
    color    = src[0]
    bits     = src[1] | (src[2] << 8)
    num_bits = 16
    data_pos = 3
    repeat_mode  = False
    repeat_count = 0

    pixels = [0] * (8 * height)

    def fill():
        nonlocal bits, num_bits, data_pos
        if num_bits <= 8:
            if data_pos < len(src):
                bits |= src[data_pos] << num_bits
                data_pos += 1
            num_bits += 8

    def read_bits(n):
        nonlocal bits, num_bits
        fill()
        val = bits & ((1 << n) - 1)
        bits >>= n
        num_bits -= n
        return val

    for row in range(height):
        for col in range(8):
            nonlocal_color = color
            pixels[row * 8 + col] = nonlocal_color

            if not repeat_mode:
                if read_bits(1):
                    if read_bits(1):
                        diff = read_bits(3) - 4
                        if diff:
                            color = (color + diff) & 0xff
                        else:
                            repeat_mode = True
                            repeat_count = read_bits(8) - 1
                    else:
                        color = read_bits(decomp_shr)
            else:
                repeat_count -= 1
                if repeat_count == 0:
                    repeat_mode = False

    return pixels


def decode_strip_raw(src, height):
    """BMCOMP_RAW256 = 1: raw bytes, row-major."""
    result = list(src[:8 * height])
    return result


def decode_strip(src, height):
    codec = src[0]
    data  = src[1:]  # past codec byte

    decomp_shr  = codec % 10
    decomp_mask = 0xFF >> (8 - decomp_shr) if decomp_shr > 0 else 0xFF

    if codec == 1:
        return decode_strip_raw(data, height)

    elif 14 <= codec <= 18:          # ZIGZAG_V
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=False)

    elif 24 <= codec <= 28:          # ZIGZAG_H
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=True)

    elif 34 <= codec <= 38:          # ZIGZAG_VT (transparent)
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=False)

    elif 44 <= codec <= 48:          # ZIGZAG_HT (transparent)
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=True)

    elif (64 <= codec <= 68) or (84 <= codec <= 88) or \
         (104 <= codec <= 108) or (124 <= codec <= 128):   # MAJMIN variants
        return decode_strip_majmin(data, height, decomp_shr)

    else:
        print(f"  WARNING: unsupported codec {codec} ({hex(codec)}) — filling magenta",
              file=sys.stderr)
        return [255] * (8 * height)


# ---------------------------------------------------------------------------
# Main decoder
# ---------------------------------------------------------------------------

def decode_room(lflf_dir, out_path):
    room_dir = os.path.join(lflf_dir, 'ROOM')

    # --- RMHD: room dimensions ---
    rmhd   = open(os.path.join(room_dir, 'RMHD'), 'rb').read()
    width  = le16(rmhd, 8)
    height = le16(rmhd, 10)
    print(f"  Room: {width} x {height}")

    # --- CLUT: 256-colour palette ---
    clut    = open(os.path.join(room_dir, 'CLUT'), 'rb').read()
    pal_raw = clut[8:]
    palette = [(pal_raw[i*3], pal_raw[i*3+1], pal_raw[i*3+2]) for i in range(256)]

    # --- RMIM: compressed image ---
    rmim = open(os.path.join(room_dir, 'RMIM'), 'rb').read()

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
        print(f"  e.g. {sys.argv[0]} game/monkey1/gen/full_dump/DISK_0001/LECF/LFLF_0028 out.png")
        sys.exit(1)

    decode_room(sys.argv[1], sys.argv[2])
