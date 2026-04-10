#!/usr/bin/env python3
"""
Decode SCUMM v5 object images (OBIM) to PNG.
Usage: python3 decode_object.py <obim_file> <output.png>
"""

import struct
import sys
from PIL import Image

def be32(data, pos):
    return struct.unpack_from('>I', data, pos)[0]

def le32(data, pos):
    return struct.unpack_from('<I', data, pos)[0]

def le16(data, pos):
    return struct.unpack_from('<H', data, pos)[0]

def find_block(data, start, end, tag):
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

def decode_strip_raw(src, height):
    """Codec 1: raw uncompressed pixels"""
    return list(src[:8 * height])

def decode_strip_basic(src, height, decomp_shr, decomp_mask, horizontal):
    """Codecs 14-18, 24-28, 34-38, 44-48: bit-delta encoding"""
    pos   = 0
    color = src[pos]; pos += 1
    bits  = src[pos]; pos += 1
    cl    = 8
    inc   = -1
    pixels = [0] * (8 * height)

    def fill_bits():
        nonlocal bits, cl, pos
        if cl <= 8:
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
        for row in range(height):
            for col in range(8):
                fill_bits()
                pixels[row * 8 + col] = color
                if not read_bit():
                    pass
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
        for col in range(8):
            for row in range(height):
                fill_bits()
                pixels[row * 8 + col] = color
                if not read_bit():
                    pass
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
    """Codecs 64-68, 84-88, 104-108, 124-128: MajMinCodec (from ScummVM gfx.cpp)"""
    color        = src[0]
    bits         = src[1] | (src[2] << 8)
    num_bits     = 16
    data_pos     = 3
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

def decode_strip(src, height):
    """Decode a single strip based on codec byte"""
    codec = src[0]
    data  = src[1:]
    decomp_shr  = codec % 10
    decomp_mask = 0xFF >> (8 - decomp_shr) if decomp_shr > 0 else 0xFF

    if codec == 1:
        return decode_strip_raw(data, height)
    elif 14 <= codec <= 18:
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=False)
    elif 24 <= codec <= 28:
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=True)
    elif 34 <= codec <= 38:
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=False)
    elif 44 <= codec <= 48:
        return decode_strip_basic(data, height, decomp_shr, decomp_mask, horizontal=True)
    elif (64 <= codec <= 68) or (84 <= codec <= 88) or \
         (104 <= codec <= 108) or (124 <= codec <= 128):
        return decode_strip_majmin(data, height, decomp_shr)
    else:
        raise ValueError(f"Unsupported codec: {codec}")

def decode_object(obim_path, out_path, room_dir=None):
    """Decode OBIM file to PNG

    Args:
        obim_path: Path to OBIM file
        out_path: Output PNG path
        room_dir: Optional path to room directory containing CLUT for palette
    """
    obim = open(obim_path, 'rb').read()

    # OBIM structure: OBIM tag, then IMHD (header), IMAG (image data)
    if obim[:4] != b'OBIM':
        raise ValueError("Not an OBIM file")

    obim_size = be32(obim, 4)

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
        import os
        clut_path = os.path.join(room_dir, 'CLUT')
        if os.path.exists(clut_path):
            clut = open(clut_path, 'rb').read()
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
