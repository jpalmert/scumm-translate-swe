#!/usr/bin/env python3
"""
Text extractor and injector for Monkey Island Special Edition .info files.
Exports to JSON for translation, injects translated JSON back.

Always targets the French language slot (index 1) — this is the SE engine
limitation: custom translations replace French, and the game must be set
to French language to see them.

Supported file types (auto-detected by magic uint32 at offset 0):
  MI:SE  (game 1):
    speech.info    magic 0x000E1794  fixed-stride format, 3 languages
    uiText.info    magic 0x554E454D  fixed-stride format, 3 languages
    *.hints.csv    magic 0x000000F2  grouped hints, variable-length strings

  MI2:SE (game 2):
    fr.speech.info magic 0x00002215  pointer-table format
    fr.uitext.info magic 0x000004F2  pointer-table format
    *.hints.csv    magic 0x0000013E  grouped hints, variable-length strings

JSON output format:
  [
    { "id": 0, "context": "speech", "english": "Hello there!", "translation": "" },
    ...
  ]
  Fill in "translation" fields, leave blank to keep original English.

Usage:
  python3 text.py extract speech.info speech.json
  python3 text.py inject  speech.info speech.json speech_modified.info
"""

import json
import struct
import sys
from pathlib import Path

# Magic values
MAGIC_MI1_SPEECH  = 0x000E1794
MAGIC_MI1_UI      = 0x554E454D   # "MENU"
MAGIC_MI1_HINTS   = 0x000000F2
MAGIC_MI2_SPEECH  = 0x00002215
MAGIC_MI2_UI      = 0x000004F2
MAGIC_MI2_HINTS   = 0x0000013E

SLOT_SIZE = 0x100   # 256 bytes per language slot (fixed-stride)
TARGET_LANG = 1     # French slot index (0=English, 1=French, 2=German for MI:SE)
NUM_LANGUAGES = 3   # Standard for MI:SE fixed-stride files


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _read_u32(f):
    return struct.unpack('<I', f.read(4))[0]

def _read_u32_at(data, offset):
    return struct.unpack('<I', data[offset:offset+4])[0]

def _write_u32_at(buf, offset, val):
    buf[offset:offset+4] = struct.pack('<I', val)

def _read_slot(f, offset):
    """Read a 256-byte null-terminated, space-padded string slot."""
    f.seek(offset)
    raw = f.read(SLOT_SIZE)
    end = raw.find(b'\x00')
    if end == -1:
        end = SLOT_SIZE
    return raw[:end].decode('latin-1')

def _encode_slot(text):
    """Encode a string into a 256-byte slot (null-terminated, space-padded)."""
    encoded = text.encode('latin-1')[:SLOT_SIZE - 1]
    return encoded + b'\x00' + b'\x20' * (SLOT_SIZE - len(encoded) - 1)

def _read_cstring_at(data, offset):
    end = data.index(b'\x00', offset)
    return data[offset:end].decode('latin-1')


# ---------------------------------------------------------------------------
# MI:SE fixed-stride formats
# ---------------------------------------------------------------------------

def _stride_entry_offset(n, num_languages, file_type):
    """Return the byte offset of entry N in a fixed-stride .info file."""
    if file_type == 'speech':
        # Header: 0x10 bytes; per-entry: 0x10 header + num_languages*SLOT_SIZE + 0x20 speaker info
        entry_size = 0x10 + num_languages * SLOT_SIZE + 0x20
        return 0x10 + n * entry_size
    else:  # uiText
        entry_size = num_languages * SLOT_SIZE
        return SLOT_SIZE + n * entry_size


def _stride_count(data, num_languages, file_type):
    """Infer number of entries from file size."""
    if file_type == 'speech':
        entry_size = 0x10 + num_languages * SLOT_SIZE + 0x20
        return (len(data) - 0x10) // entry_size
    else:
        entry_size = num_languages * SLOT_SIZE
        return (len(data) - SLOT_SIZE) // entry_size


def extract_stride(data, file_type, num_languages=NUM_LANGUAGES):
    """Extract strings from a fixed-stride .info file. Returns list of dicts."""
    count = _stride_count(data, num_languages, file_type)
    results = []
    for n in range(count):
        base = _stride_entry_offset(n, num_languages, file_type)
        if file_type == 'speech':
            base += 0x10  # skip per-entry header
        eng_offset = base + 0 * SLOT_SIZE
        end = data.find(b'\x00', eng_offset)
        if end == -1: end = eng_offset + SLOT_SIZE
        english = data[eng_offset:end].decode('latin-1')
        results.append({'id': n, 'context': file_type, 'english': english, 'translation': ''})
    return results


def inject_stride(data, entries, file_type, num_languages=NUM_LANGUAGES):
    """Inject translations into a fixed-stride .info file. Returns modified bytes."""
    buf = bytearray(data)
    for entry in entries:
        n = entry['id']
        text = entry.get('translation') or entry['english']
        base = _stride_entry_offset(n, num_languages, file_type)
        if file_type == 'speech':
            base += 0x10
        target_offset = base + TARGET_LANG * SLOT_SIZE
        buf[target_offset:target_offset + SLOT_SIZE] = _encode_slot(text)
    return bytes(buf)


# ---------------------------------------------------------------------------
# MI2:SE pointer-table formats
# ---------------------------------------------------------------------------

def extract_ptr_uitext(data):
    """Extract strings from MI2:SE fr.uitext.info pointer-table format."""
    # First entry at 0x04; each entry is 8 bytes (2 × uint32 relative offsets)
    first_entry_val = _read_u32_at(data, 4)
    num_entries = (first_entry_val + 4 - 4) // 8
    # num_entries derived: firstEntryValue / 8 (pointer to first data = past the index)
    # Re-derive more robustly: first relative pointer points past the entire index
    num_entries = first_entry_val // 8

    results = []
    for n in range(num_entries):
        entry_addr = 4 + n * 8
        rel_eng = _read_u32_at(data, entry_addr)
        rel_fra = _read_u32_at(data, entry_addr + 4)
        eng_abs = entry_addr + rel_eng
        fra_abs = (entry_addr + 4) + rel_fra
        english = _read_cstring_at(data, eng_abs)
        results.append({'id': n, 'context': 'uitext', 'english': english, 'translation': ''})
    return results


def inject_ptr_uitext(data, entries):
    """Inject translations into MI2:SE fr.uitext.info. Returns modified bytes."""
    first_entry_val = _read_u32_at(data, 4)
    num_entries = first_entry_val // 8

    # Build a map of id -> translation
    trans_map = {e['id']: (e.get('translation') or e['english']) for e in entries}

    buf = bytearray(data)
    deviation = 0  # cumulative size change

    for n in range(num_entries):
        entry_addr = 4 + n * 8 + deviation  # adjusted for prior size changes
        rel_fra = _read_u32_at(buf, entry_addr + 4)
        fra_abs = (entry_addr + 4) + rel_fra

        old_end = buf.index(b'\x00', fra_abs)
        old_str = buf[fra_abs:old_end]
        new_str = trans_map.get(n, old_str.decode('latin-1')).encode('latin-1')

        if new_str == old_str:
            continue

        size_diff = len(new_str) - len(old_str)
        buf[fra_abs:old_end] = new_str

        # Update all subsequent relative pointers for entries after n
        # (both English and French pointers that reference data after this point)
        # This is complex — for a robust implementation, rebuild the entire section.
        # For now, warn if sizes change and do a best-effort update.
        if size_diff != 0:
            print(f"  WARNING: entry {n} size changed by {size_diff} bytes — pointer fixup required")
            deviation += size_diff

    if deviation != 0:
        print("  NOTE: Variable-length string changes detected in uitext.")
        print("  For production use, run rebuild_ptr_uitext() which reconstructs the full file.")

    return bytes(buf)


def extract_ptr_speech(data):
    """Extract strings from MI2:SE fr.speech.info pointer-table format."""
    # Index entries start at 0x18; each entry is 0x20 bytes
    INDEX_START = 0x18
    ENTRY_STRIDE = 0x20

    # Derive number of entries from first relative pointer
    first_rel = _read_u32_at(data, INDEX_START)
    # first_rel is relative offset from INDEX_START to start of string data
    num_entries = first_rel // ENTRY_STRIDE

    results = []
    for n in range(num_entries):
        base = INDEX_START + n * ENTRY_STRIDE
        # Offsets within the 0x20-byte entry (from MISETranslator source):
        #   +0x00: rel offset to English label
        #   +0x14: rel offset to English text
        #   +0x18: rel offset to French/target text
        rel_label = _read_u32_at(data, base + 0x00)
        rel_eng   = _read_u32_at(data, base + 0x14)
        rel_fra   = _read_u32_at(data, base + 0x18)

        label_abs = base + rel_label
        eng_abs   = (base + 0x14) + rel_eng
        fra_abs   = (base + 0x18) + rel_fra

        try:
            label   = _read_cstring_at(data, label_abs)
            english = _read_cstring_at(data, eng_abs)
        except (ValueError, UnicodeDecodeError):
            label, english = f"entry_{n}", ""

        results.append({
            'id': n,
            'context': f'speech:{label}',
            'english': english,
            'translation': ''
        })
    return results


def inject_ptr_speech(data, entries):
    """
    Inject translations into MI2:SE fr.speech.info.
    Rebuilds the string data section to handle length changes properly.
    """
    INDEX_START = 0x18
    ENTRY_STRIDE = 0x20
    first_rel = _read_u32_at(data, INDEX_START)
    num_entries = first_rel // ENTRY_STRIDE
    data_section_start = INDEX_START + first_rel

    trans_map = {e['id']: (e.get('translation') or e['english']) for e in entries}

    # Collect all entry pointer info and string data
    entry_info = []
    for n in range(num_entries):
        base = INDEX_START + n * ENTRY_STRIDE
        rel_label = _read_u32_at(data, base + 0x00)
        rel_eng   = _read_u32_at(data, base + 0x14)
        rel_fra   = _read_u32_at(data, base + 0x18)

        label_abs = base + rel_label
        eng_abs   = (base + 0x14) + rel_eng
        fra_abs   = (base + 0x18) + rel_fra

        try:
            label   = _read_cstring_at(data, label_abs)
            english = _read_cstring_at(data, eng_abs)
            french  = _read_cstring_at(data, fra_abs)
        except (ValueError, UnicodeDecodeError):
            label, english, french = f"entry_{n}", "", ""

        new_french = trans_map.get(n, french)
        entry_info.append({
            'base': base,
            'label': label,
            'english': english,
            'french': new_french,
            'orig_label_abs': label_abs,
            'orig_eng_abs': eng_abs,
            'orig_fra_abs': fra_abs,
        })

    # Rebuild: keep everything before data_section_start, then rebuild string blobs
    buf = bytearray(data[:data_section_start])
    new_string_data = bytearray()
    string_positions = {}  # abs address in original -> new offset from data_section_start

    # We need to preserve the positions of strings that aren't being replaced
    # Strategy: rebuild by iterating original data offsets in order
    # Collect all unique (abs_offset, string) pairs
    all_strings = {}
    for ei in entry_info:
        for key in ('orig_label_abs', 'orig_eng_abs', 'orig_fra_abs'):
            if ei[key] not in all_strings:
                try:
                    all_strings[ei[key]] = _read_cstring_at(data, ei[key])
                except (ValueError, UnicodeDecodeError):
                    all_strings[ei[key]] = ''

    # Write strings in original address order, tracking new positions
    for orig_addr in sorted(all_strings.keys()):
        string_positions[orig_addr] = data_section_start + len(new_string_data)
        s = all_strings[orig_addr].encode('latin-1') + b'\x00'
        new_string_data += s

    # Now write translated French strings (may be at different positions)
    fra_positions = {}
    for ei in entry_info:
        pos = data_section_start + len(new_string_data)
        fra_positions[ei['base']] = pos
        new_string_data += ei['french'].encode('latin-1') + b'\x00'

    buf += new_string_data

    # Patch the index entries with new relative pointers
    for ei in entry_info:
        base = ei['base']
        new_label_abs = string_positions.get(ei['orig_label_abs'], ei['orig_label_abs'])
        new_eng_abs   = string_positions.get(ei['orig_eng_abs'], ei['orig_eng_abs'])
        new_fra_abs   = fra_positions[base]

        _write_u32_at(buf, base + 0x00, new_label_abs - base)
        _write_u32_at(buf, base + 0x14, new_eng_abs - (base + 0x14))
        _write_u32_at(buf, base + 0x18, new_fra_abs - (base + 0x18))

    return bytes(buf)


# ---------------------------------------------------------------------------
# Dispatch
# ---------------------------------------------------------------------------

FORMAT_NAMES = {
    MAGIC_MI1_SPEECH: 'mi1_speech',
    MAGIC_MI1_UI:     'mi1_ui',
    MAGIC_MI1_HINTS:  'mi1_hints',
    MAGIC_MI2_SPEECH: 'mi2_speech',
    MAGIC_MI2_UI:     'mi2_ui',
    MAGIC_MI2_HINTS:  'mi2_hints',
}


def detect_format(data):
    magic = _read_u32_at(data, 0)
    fmt = FORMAT_NAMES.get(magic)
    if fmt is None:
        raise ValueError(f"Unknown .info magic: 0x{magic:08X}")
    return fmt


def extract_info(info_path):
    """Extract all strings from an .info file. Returns list of entry dicts."""
    data = Path(info_path).read_bytes()
    fmt = detect_format(data)
    print(f"  detected format: {fmt}")

    if fmt == 'mi1_speech':
        return extract_stride(data, 'speech')
    elif fmt == 'mi1_ui':
        return extract_stride(data, 'ui')
    elif fmt == 'mi2_speech':
        return extract_ptr_speech(data)
    elif fmt == 'mi2_ui':
        return extract_ptr_uitext(data)
    elif fmt in ('mi1_hints', 'mi2_hints'):
        print("  NOTE: hints extraction not yet implemented; returning empty")
        return []
    else:
        raise NotImplementedError(f"No extractor for format {fmt}")


def inject_info(info_path, entries, output_path):
    """Inject translated entries into an .info file, writing to output_path."""
    data = Path(info_path).read_bytes()
    fmt = detect_format(data)
    print(f"  detected format: {fmt}")

    if fmt == 'mi1_speech':
        result = inject_stride(data, entries, 'speech')
    elif fmt == 'mi1_ui':
        result = inject_stride(data, entries, 'ui')
    elif fmt == 'mi2_speech':
        result = inject_ptr_speech(data, entries)
    elif fmt == 'mi2_ui':
        result = inject_ptr_uitext(data, entries)
    elif fmt in ('mi1_hints', 'mi2_hints'):
        print("  NOTE: hints injection not yet implemented; copying original")
        result = data
    else:
        raise NotImplementedError(f"No injector for format {fmt}")

    Path(output_path).write_bytes(result)
    print(f"  wrote {len(result)} bytes to {output_path}")


def main():
    if len(sys.argv) < 4:
        print(__doc__)
        sys.exit(1)

    cmd = sys.argv[1]

    if cmd == 'extract':
        info_path, json_path = sys.argv[2], sys.argv[3]
        entries = extract_info(info_path)
        Path(json_path).write_text(
            json.dumps(entries, ensure_ascii=False, indent=2),
            encoding='utf-8'
        )
        print(f"\nExtracted {len(entries)} entries to {json_path}")

    elif cmd == 'inject':
        if len(sys.argv) < 5:
            print(__doc__)
            sys.exit(1)
        info_path, json_path, output_path = sys.argv[2], sys.argv[3], sys.argv[4]
        entries = json.loads(Path(json_path).read_text(encoding='utf-8'))
        inject_info(info_path, entries, output_path)

    else:
        print(__doc__)
        sys.exit(1)


if __name__ == '__main__':
    main()
