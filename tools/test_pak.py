"""Tests for pak.py — PAK archive extract/repack round-trip."""

import sys
import os
import struct
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from pak import extract, repack, detect_game, MAGIC, MAGIC_GOG, HEADER_SIZE, ENTRY_SIZE


def _build_pak(magic, files):
    """
    Build a minimal PAK archive in memory.

    files: list of (name: str, data: bytes)
    Returns bytes of the complete PAK.
    """
    num_files = len(files)

    # Build names blob (null-terminated strings)
    names_blob = b''
    name_offsets = []
    for name, _ in files:
        name_offsets.append(len(names_blob))
        names_blob += name.encode('ascii') + b'\x00'

    # Build data blob (files concatenated)
    data_blob = b''
    data_offsets = []
    for _, data in files:
        data_offsets.append(len(data_blob))
        data_blob += data

    # Layout:
    #   Header (0x28)
    #   Index (empty for our test — 0 bytes)
    #   File entries
    #   File names
    #   File data
    start_of_index = HEADER_SIZE
    size_of_index = 0
    start_of_entries = start_of_index + size_of_index
    size_of_entries = num_files * ENTRY_SIZE
    start_of_names = start_of_entries + size_of_entries
    size_of_names = len(names_blob)
    start_of_data = start_of_names + size_of_names
    size_of_data = len(data_blob)

    # Header
    hdr = magic
    hdr += struct.pack('<I', 1)              # version
    hdr += struct.pack('<I', start_of_index)
    hdr += struct.pack('<I', start_of_entries)
    hdr += struct.pack('<I', start_of_names)
    hdr += struct.pack('<I', start_of_data)
    hdr += struct.pack('<I', size_of_index)
    hdr += struct.pack('<I', size_of_entries)
    hdr += struct.pack('<I', size_of_names)
    hdr += struct.pack('<I', size_of_data)

    # File entries
    entries = b''
    for i in range(num_files):
        entries += struct.pack('<I', data_offsets[i])   # data_pos
        entries += struct.pack('<I', name_offsets[i])    # name_pos
        entries += struct.pack('<I', len(files[i][1]))   # data_size
        entries += struct.pack('<I', len(files[i][1]))   # data_size2
        entries += struct.pack('<I', 0)                  # compressed

    return hdr + entries + names_blob + data_blob


class TestDetectGame(unittest.TestCase):
    """Test game detection from PAK filename."""

    def test_monkey1(self):
        self.assertEqual(detect_game("Monkey1.pak"), 1)

    def test_monkey2(self):
        self.assertEqual(detect_game("Monkey2.pak"), 2)

    def test_mi2_lowercase(self):
        self.assertEqual(detect_game("/path/to/mi2_special.pak"), 2)

    def test_unknown_defaults_to_1(self):
        self.assertEqual(detect_game("game.pak"), 1)


class TestPakRoundTrip(unittest.TestCase):
    """Test extract-then-repack preserves file contents."""

    def _write_pak(self, magic, files):
        """Write a synthetic PAK to a temp file, return its path."""
        data = _build_pak(magic, files)
        fd, path = tempfile.mkstemp(suffix='.pak', dir=os.environ.get('TMPDIR', '/tmp'))
        os.write(fd, data)
        os.close(fd)
        return path

    def test_lpak_round_trip(self):
        """Extract and repack with LPAK magic preserves contents."""
        files = [
            ("file_a.txt", b"Hello World"),
            ("file_b.bin", b"\x00\x01\x02\x03"),
            ("subdir/file_c.dat", b"SCUMM data here!"),
        ]
        pak_path = self._write_pak(MAGIC, files)

        try:
            tmpdir = os.environ.get('TMPDIR', '/tmp')
            extract_dir = tempfile.mkdtemp(dir=tmpdir)
            extract(pak_path, extract_dir, game=1)

            # Verify extracted files match
            for name, expected_data in files:
                extracted = Path(extract_dir) / name
                self.assertTrue(extracted.exists(), f"{name} not extracted")
                self.assertEqual(extracted.read_bytes(), expected_data)

            # Repack and verify round-trip
            repacked_path = os.path.join(tmpdir, "repacked.pak")
            repack(extract_dir, repacked_path, pak_path, game=1)

            # Extract repacked and compare
            extract_dir2 = tempfile.mkdtemp(dir=tmpdir)
            extract(repacked_path, extract_dir2, game=1)

            for name, expected_data in files:
                extracted = Path(extract_dir2) / name
                self.assertTrue(extracted.exists(), f"{name} not in repacked")
                self.assertEqual(extracted.read_bytes(), expected_data)
        finally:
            os.unlink(pak_path)

    def test_kapl_magic_preserved(self):
        """GOG PAK (KAPL magic) is accepted and preserved through repack."""
        files = [("test.txt", b"GOG version")]
        pak_path = self._write_pak(MAGIC_GOG, files)

        try:
            tmpdir = os.environ.get('TMPDIR', '/tmp')
            extract_dir = tempfile.mkdtemp(dir=tmpdir)
            extract(pak_path, extract_dir, game=1)

            repacked_path = os.path.join(tmpdir, "repacked_gog.pak")
            repack(extract_dir, repacked_path, pak_path, game=1)

            # Verify magic bytes preserved
            with open(repacked_path, 'rb') as f:
                magic = f.read(4)
            self.assertEqual(magic, MAGIC_GOG)
        finally:
            os.unlink(pak_path)

    def test_invalid_magic_raises(self):
        """Non-PAK file raises ValueError."""
        tmpdir = os.environ.get('TMPDIR', '/tmp')
        fd, path = tempfile.mkstemp(suffix='.pak', dir=tmpdir)
        os.write(fd, b'NOPE' + b'\x00' * 36)
        os.close(fd)

        try:
            with self.assertRaises(ValueError):
                extract(path, tempfile.mkdtemp(dir=tmpdir), game=1)
        finally:
            os.unlink(path)

    def test_modified_file_in_repack(self):
        """Repack picks up modified files from input dir."""
        original_data = b"original"
        modified_data = b"modified content that is longer"
        files = [("data.bin", original_data)]
        pak_path = self._write_pak(MAGIC, files)

        try:
            tmpdir = os.environ.get('TMPDIR', '/tmp')
            extract_dir = tempfile.mkdtemp(dir=tmpdir)
            extract(pak_path, extract_dir, game=1)

            # Modify the extracted file
            (Path(extract_dir) / "data.bin").write_bytes(modified_data)

            repacked_path = os.path.join(tmpdir, "repacked_mod.pak")
            repack(extract_dir, repacked_path, pak_path, game=1)

            # Extract repacked and verify modification
            extract_dir2 = tempfile.mkdtemp(dir=tmpdir)
            extract(repacked_path, extract_dir2, game=1)

            result = (Path(extract_dir2) / "data.bin").read_bytes()
            self.assertEqual(result, modified_data)
        finally:
            os.unlink(pak_path)


if __name__ == "__main__":
    unittest.main()
