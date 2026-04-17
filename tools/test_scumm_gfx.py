"""Tests for scumm_gfx.py — block parsing and binary read utilities."""

import sys
import os
import struct
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from scumm_gfx import find_block, be32, le32, le16


class TestBe32(unittest.TestCase):
    """Test big-endian 32-bit read."""

    def test_zero(self):
        self.assertEqual(be32(b'\x00\x00\x00\x00', 0), 0)

    def test_one(self):
        self.assertEqual(be32(b'\x00\x00\x00\x01', 0), 1)

    def test_known_value(self):
        # 0x01020304 = 16909060
        self.assertEqual(be32(b'\x01\x02\x03\x04', 0), 0x01020304)

    def test_max_value(self):
        self.assertEqual(be32(b'\xff\xff\xff\xff', 0), 0xFFFFFFFF)

    def test_with_offset(self):
        data = b'\xAA\xBB\x00\x00\x00\x2A'
        self.assertEqual(be32(data, 2), 0x0000002A)


class TestLe32(unittest.TestCase):
    """Test little-endian 32-bit read."""

    def test_zero(self):
        self.assertEqual(le32(b'\x00\x00\x00\x00', 0), 0)

    def test_one(self):
        self.assertEqual(le32(b'\x01\x00\x00\x00', 0), 1)

    def test_known_value(self):
        # LE: 0x04030201 stored as 01 02 03 04
        self.assertEqual(le32(b'\x01\x02\x03\x04', 0), 0x04030201)

    def test_with_offset(self):
        data = b'\xFF\xFF\x2A\x00\x00\x00'
        self.assertEqual(le32(data, 2), 0x0000002A)


class TestLe16(unittest.TestCase):
    """Test little-endian 16-bit read."""

    def test_zero(self):
        self.assertEqual(le16(b'\x00\x00', 0), 0)

    def test_one(self):
        self.assertEqual(le16(b'\x01\x00', 0), 1)

    def test_known_value(self):
        self.assertEqual(le16(b'\x00\x01', 0), 256)

    def test_max_value(self):
        self.assertEqual(le16(b'\xff\xff', 0), 0xFFFF)

    def test_with_offset(self):
        data = b'\xAA\x34\x12'
        self.assertEqual(le16(data, 1), 0x1234)


class TestFindBlock(unittest.TestCase):
    """Test SCUMM block finding in binary data."""

    def _make_block(self, tag, payload=b''):
        """Create a SCUMM block: 4-byte tag + 4-byte BE size + payload."""
        size = 8 + len(payload)
        return tag.encode('ascii') + struct.pack('>I', size) + payload

    def test_find_single_block(self):
        block = self._make_block('RMHD', b'\x00' * 4)
        pos = find_block(block, 0, len(block), 'RMHD')
        self.assertEqual(pos, 0)

    def test_find_second_block(self):
        b1 = self._make_block('RMHD', b'\x00' * 4)
        b2 = self._make_block('CLUT', b'\xFF' * 8)
        data = b1 + b2
        pos = find_block(data, 0, len(data), 'CLUT')
        self.assertEqual(pos, len(b1))

    def test_block_not_found(self):
        block = self._make_block('RMHD', b'\x00' * 4)
        pos = find_block(block, 0, len(block), 'CLUT')
        self.assertEqual(pos, -1)

    def test_finds_first_occurrence(self):
        b1 = self._make_block('SMAP', b'\x01')
        b2 = self._make_block('SMAP', b'\x02')
        data = b1 + b2
        pos = find_block(data, 0, len(data), 'SMAP')
        self.assertEqual(pos, 0)

    def test_search_with_start_offset(self):
        b1 = self._make_block('SMAP', b'\x01')
        b2 = self._make_block('SMAP', b'\x02')
        data = b1 + b2
        # Start search after first block
        pos = find_block(data, len(b1), len(data), 'SMAP')
        self.assertEqual(pos, len(b1))

    def test_empty_data(self):
        pos = find_block(b'', 0, 0, 'RMHD')
        self.assertEqual(pos, -1)

    def test_data_too_short_for_header(self):
        pos = find_block(b'RMH', 0, 3, 'RMHD')
        self.assertEqual(pos, -1)

    def test_block_with_zero_size_breaks(self):
        # A block with size < 8 should cause the search to stop
        bad = b'RMHD' + struct.pack('>I', 0)
        pos = find_block(bad, 0, len(bad), 'RMHD')
        # The tag matches at position 0, so it's found before size is used for skipping
        self.assertEqual(pos, 0)

    def test_block_with_zero_size_stops_search_for_other(self):
        # Searching for a different tag past a zero-size block should stop
        bad = b'RMHD' + struct.pack('>I', 4)  # size < 8
        more = self._make_block('CLUT')
        data = bad + more
        pos = find_block(data, 0, len(data), 'CLUT')
        self.assertEqual(pos, -1)


if __name__ == "__main__":
    unittest.main()
