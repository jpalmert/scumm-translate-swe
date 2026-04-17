"""
Shared SCUMM v5 graphics decoding helpers.

Block parsing, strip decoders, and binary read utilities used by
decode_room.py and decode_object.py.

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
"""

import struct
import sys


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

    Macros from ScummVM gfx.cpp:
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
    """
    MajMinCodec::decodeLine -- used for BMCOMP_MAJMIN_H* and RMAJMIN_H*.

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
            pixels[row * 8 + col] = color

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
    return list(src[:8 * height])


def decode_strip(src, height):
    """Decode a single strip based on codec byte."""
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
        print(f"  WARNING: unsupported codec {codec} ({hex(codec)}) -- filling magenta",
              file=sys.stderr)
        return [255] * (8 * height)
