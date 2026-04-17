"""Tests for calc_padding.py — encode_swedish() and scumm_byte_len()."""

import sys
import os
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from calc_padding import encode_swedish, scumm_byte_len


class TestEncodeSwedish(unittest.TestCase):
    """Test Swedish character encoding to SCUMM escape codes."""

    def test_a_ring(self):
        self.assertEqual(encode_swedish("å"), "\\123")

    def test_a_umlaut(self):
        self.assertEqual(encode_swedish("ä"), "\\124")

    def test_o_umlaut(self):
        self.assertEqual(encode_swedish("ö"), "\\125")

    def test_A_ring(self):
        self.assertEqual(encode_swedish("Å"), "\\091")

    def test_A_umlaut(self):
        self.assertEqual(encode_swedish("Ä"), "\\092")

    def test_O_umlaut(self):
        self.assertEqual(encode_swedish("Ö"), "\\093")

    def test_e_acute(self):
        self.assertEqual(encode_swedish("é"), "\\130")

    def test_non_swedish_unchanged(self):
        self.assertEqual(encode_swedish("hello"), "hello")

    def test_mixed_text(self):
        self.assertEqual(encode_swedish("bår"), "b\\123r")

    def test_multiple_swedish_chars(self):
        self.assertEqual(encode_swedish("åäö"), "\\123\\124\\125")

    def test_empty_string(self):
        self.assertEqual(encode_swedish(""), "")

    def test_parenthesized_prefix_stripped(self):
        # encode_swedish strips (xxx) prefix used in scummtr headers
        self.assertEqual(encode_swedish("(4)mugg"), "mugg")

    def test_parenthesized_prefix_with_swedish(self):
        self.assertEqual(encode_swedish("(2)öl"), "\\125l")

    def test_no_paren_prefix_if_not_at_start(self):
        # Only strips if starts with (
        self.assertEqual(encode_swedish("a(2)b"), "a(2)b")

    def test_registered_trademark(self):
        self.assertEqual(encode_swedish("®"), "\\015")

    def test_e_circumflex(self):
        self.assertEqual(encode_swedish("ê"), "\\136")


class TestScummByteLen(unittest.TestCase):
    """Test SCUMM string byte length calculation."""

    def test_plain_ascii(self):
        self.assertEqual(scumm_byte_len("hello"), 5)

    def test_single_escape(self):
        # \091 is a 4-char escape sequence representing 1 byte
        self.assertEqual(scumm_byte_len("ab\\091c"), 4)

    def test_multiple_escapes(self):
        self.assertEqual(scumm_byte_len("\\091\\092\\093"), 3)

    def test_empty_string(self):
        self.assertEqual(scumm_byte_len(""), 0)

    def test_escape_at_end(self):
        self.assertEqual(scumm_byte_len("ab\\123"), 3)

    def test_escape_at_start(self):
        self.assertEqual(scumm_byte_len("\\124xy"), 3)

    def test_backslash_not_followed_by_three_digits(self):
        # \ab is not a valid escape — counts as 3 chars
        self.assertEqual(scumm_byte_len("\\ab"), 3)

    def test_backslash_with_only_two_digits(self):
        # \09 at end — not enough digits, counts individually
        self.assertEqual(scumm_byte_len("\\09"), 3)

    def test_mixed_escapes_and_text(self):
        # "hej\123r" = h(1) e(1) j(1) \123(1) r(1) = 5
        self.assertEqual(scumm_byte_len("hej\\123r"), 5)

    def test_at_padding(self):
        self.assertEqual(scumm_byte_len("mugg@@@@"), 8)


if __name__ == "__main__":
    unittest.main()
