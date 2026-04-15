#!/usr/bin/env python3
"""
Find all dynamic object name replacements in MI1 SCUMM scripts.

Extracts script blocks with scummrp, decompiles them with descumm, and
parses the output to build a mapping of which scummtr lines replace which
OBNA lines at runtime.

Output: game/<game>/gen/dynamic_names.json

The JSON maps OBNA line numbers to lists of replacement line numbers:

    {
      "95": [96, 97, 98, 99, 100],
      "538": [544, 545],
      ...
    }

Line numbers are 1-based (matching text editor line numbers in swedish.txt).
At build time, calc_padding.py reads the Swedish text at these line numbers
and pads the OBNA to fit the longest replacement.

Usage:
    python3 tools/find_dynamic_names.py <game_dir> [output_json]

Requires: scummrp and descumm in bin/<platform>/
"""

import json
import os
import platform
import re
import shutil
import struct
import subprocess
import sys
import tempfile


def find_bin(name):
    plat = "darwin" if platform.system() == "Darwin" else "linux"
    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    path = os.path.join(repo, "bin", plat, name)
    if os.path.isfile(path):
        return path
    raise FileNotFoundError(f"{name} not found at {path}")


def extract_scripts(scummrp, game_dir, tmp_dir, block_type):
    dump_dir = os.path.join(tmp_dir, block_type)
    os.makedirs(dump_dir, exist_ok=True)
    subprocess.run(
        [scummrp, "-g", "monkeycdalt", "-p", game_dir, "-t", block_type, "-o", "-d", dump_dir],
        capture_output=True, check=True,
    )
    return dump_dir


def decompile(descumm, script_path):
    try:
        result = subprocess.run(
            [descumm, "-5", script_path],
            capture_output=True, timeout=30,
        )
        return result.stdout.decode("latin-1", errors="replace")
    except Exception:
        return ""


def extract_verb_from_obcd(obcd_path):
    with open(obcd_path, "rb") as f:
        data = f.read()
    obj_id = None
    pos = data.find(b"CDHD")
    if pos >= 0:
        obj_id = struct.unpack("<H", data[pos + 8 : pos + 10])[0]
    pos = data.find(b"VERB")
    if pos < 0:
        return None, obj_id
    size = struct.unpack(">I", data[pos + 4 : pos + 8])[0]
    return data[pos : pos + size], obj_id


RE_SET_OBJECT_NAME = re.compile(
    r'setObjectName\((\d+|VAR_ME|Local\[\d+\]|VAR_\w+)\s*,'
)


def find_setobjectname_targets(text, verb_obj_id=None):
    """Find setObjectName calls in descumm output. Returns list of (target_obj_id_or_None, line_index).
    line_index is the 0-based position of the string within the decompiled output's string list."""
    targets = []
    # descumm outputs strings in order — we need to count which string each
    # setObjectName corresponds to in the scummtr extraction order.
    # The string index in descumm corresponds to the scummtr line index for that block.
    #
    # Count all inline strings (anything in quotes) to track position.
    string_index = -1
    for line in text.split("\n"):
        # Count quoted strings to track position
        if '"' in line:
            string_index += 1

        m = RE_SET_OBJECT_NAME.search(line)
        if m:
            target_raw = m.group(1)
            if target_raw == "VAR_ME" and verb_obj_id is not None:
                targets.append((verb_obj_id, string_index))
            else:
                try:
                    targets.append((int(target_raw), string_index))
                except ValueError:
                    pass  # variable target — can't resolve
    return targets


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <game_dir> [output_json]", file=sys.stderr)
        sys.exit(1)

    game_dir = sys.argv[1]
    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    output_path = sys.argv[2] if len(sys.argv) > 2 else os.path.join(
        repo, "game", "monkey1", "gen", "dynamic_names.json"
    )

    scummrp = find_bin("scummrp")
    descumm_bin = find_bin("descumm")

    # Also extract text with scummtr to get the exact headers and line ordering
    scummtr = find_bin("scummtr")

    print("=== Extracting scummtr text (for header mapping) ===")
    tmp_dir = tempfile.mkdtemp(prefix="mi1-dynnames-")

    try:
        # Extract text with scummtr -hI (headers + opcodes) to get line order
        scummtr_out = os.path.join(tmp_dir, "text_hI.txt")
        subprocess.run(
            [scummtr, "-g", "monkeycdalt", "-p", game_dir, "-hI", "-o", "-f", scummtr_out],
            capture_output=True, check=True,
        )

        # Parse scummtr output into lines with headers, tracking:
        # - 1-based line number (for the JSON output)
        # - header occurrence index (to match descumm string positions)
        with open(scummtr_out, "r", encoding="latin-1") as f:
            all_scummtr_lines = [l.rstrip("\r\n") for l in f]

        from collections import defaultdict
        header_occurrences = defaultdict(list)  # header -> [(data_line_number_1based, text)]

        # Count only data lines (starting with [) — this matches swedish.txt
        # which has no comment lines. Line 1 = first data line.
        data_lineno = 0
        for line in all_scummtr_lines:
            if not line.startswith("["):
                continue
            data_lineno += 1
            idx = line.index("]")
            header = line[:idx+1]
            text = line[idx+1:]
            header_occurrences[header].append((data_lineno, text))

        def header_occ_to_lineno(header, occ):
            """Convert header + occurrence index to 1-based data line number."""
            entries = header_occurrences.get(header, [])
            if occ < len(entries):
                return entries[occ][0]
            return None

        print("=== Extracting and decompiling scripts ===")

        # Collect all setObjectName calls with their scummtr line references
        # Result: list of (target_obj_id, scummtr_header, occurrence_index)
        all_replacements = []

        for block_type in ["SCRP", "LSCR", "ENCD", "EXCD"]:
            print(f"  {block_type}...", end=" ", flush=True)
            dump = extract_scripts(scummrp, game_dir, tmp_dir, block_type)
            count = 0

            for root, dirs, files in os.walk(dump):
                for fname in sorted(files):
                    fpath = os.path.join(root, fname)
                    room = -1
                    for part in fpath.split(os.sep):
                        if part.startswith("LFLF_"):
                            room = int(part.split("_")[1])

                    text = decompile(descumm_bin, fpath)
                    targets = find_setobjectname_targets(text)

                    for target_obj_id, string_idx in targets:
                        if string_idx < 0:
                            continue
                        # Build the scummtr header for this block
                        if block_type == "LSCR":
                            # LSCR filename is LSCR_NNNN
                            script_num = int(fname.split("_")[1])
                            scummtr_header = f"[{room:03d}:LSCR#{script_num:04d}]"
                        elif block_type in ("ENCD", "EXCD"):
                            scummtr_header = f"[{room:03d}:{block_type}#{room:04d}]"
                        else:
                            script_num = int(fname.split("_")[1])
                            scummtr_header = f"[{room:03d}:SCRP#{script_num:04d}]"

                        all_replacements.append((target_obj_id, scummtr_header, string_idx))
                        count += 1

            print(f"{count} name changes")

        # VERB scripts (inside OBCD blocks)
        print(f"  OBCD/VERB...", end=" ", flush=True)
        obcd_dump = extract_scripts(scummrp, game_dir, tmp_dir, "OBCD")
        verb_count = 0

        for root, dirs, files in os.walk(obcd_dump):
            for fname in sorted(files):
                if not fname.startswith("OBCD_"):
                    continue
                fpath = os.path.join(root, fname)
                room = -1
                for part in fpath.split(os.sep):
                    if part.startswith("LFLF_"):
                        room = int(part.split("_")[1])

                verb_data, obj_id = extract_verb_from_obcd(fpath)
                if verb_data is None or obj_id is None:
                    continue

                verb_tmp = os.path.join(tmp_dir, "verb_tmp.bin")
                with open(verb_tmp, "wb") as f:
                    f.write(verb_data)

                text = decompile(descumm_bin, verb_tmp)
                targets = find_setobjectname_targets(text, verb_obj_id=obj_id)

                for target_obj_id, string_idx in targets:
                    if string_idx < 0:
                        continue
                    scummtr_header = f"[{room:03d}:VERB#{obj_id:04d}]"
                    all_replacements.append((target_obj_id, scummtr_header, string_idx))
                    verb_count += 1

        print(f"{verb_count} name changes")

        # Build the mapping: OBNA line number -> [replacement line numbers]
        print(f"\n=== Building mapping ===")
        mapping = {}  # obna_lineno -> [repl_linenos]

        for target_obj_id, repl_header, repl_occ in all_replacements:
            # Find the OBNA line number for this object
            obna_lineno = None
            for header in header_occurrences:
                if f"OBNA#{target_obj_id:04d}]" in header:
                    obna_lineno = header_occ_to_lineno(header, 0)
                    break

            if obna_lineno is None:
                continue

            repl_lineno = header_occ_to_lineno(repl_header, repl_occ)
            if repl_lineno is None:
                continue

            key = str(obna_lineno)
            if key not in mapping:
                mapping[key] = []
            if repl_lineno not in mapping[key]:
                mapping[key].append(repl_lineno)

        # Sort replacement lists
        for key in mapping:
            mapping[key].sort()

        output = {
            "_comment": "Maps OBNA line numbers to replacement line numbers (1-based). "
                        "Generated by tools/find_dynamic_names.py — do not edit.",
            "replacements": dict(sorted(mapping.items(), key=lambda x: int(x[0]))),
        }

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(output, f, indent=2, ensure_ascii=False)

        print(f"  OBNA lines with replacements: {len(mapping)}")
        print(f"  Total replacement references: {sum(len(v) for v in mapping.values())}")
        print(f"\n  Written: {output_path}")

    finally:
        shutil.rmtree(tmp_dir, ignore_errors=True)


if __name__ == "__main__":
    main()
