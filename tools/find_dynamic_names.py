#!/usr/bin/env python3
"""
Find all dynamic object/actor name replacements in MI1 SCUMM scripts.

Extracts script blocks with scummrp, decompiles them with descumm, and
parses the output to build a complete mapping of:
  - setObjectName calls (opcodes 0x54/0xD4): object_id -> [replacement names]
  - ActorOps Name calls (opcodes 0x13/0x93): actor_id -> [replacement names]
  - OBNA names for all objects (extracted from OBCD blocks)

Output: translation/<game>/dynamic_names.json

This only needs to be re-run if the game files change (they don't — we only
modify text, not scripts). The output is committed to the repo.

Usage:
    python3 tools/find_dynamic_names.py <game_dir> [output_json]

    game_dir:    directory containing MONKEY1.000 + MONKEY1.001
    output_json: default: translation/monkey1/dynamic_names.json

Requires: scummrp and descumm in bin/linux/ (or bin/darwin/ on macOS)
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
    """Find a tool binary in bin/<platform>/."""
    plat = "darwin" if platform.system() == "Darwin" else "linux"
    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    path = os.path.join(repo, "bin", plat, name)
    if os.path.isfile(path):
        return path
    raise FileNotFoundError(f"{name} not found at {path}")


def extract_scripts(scummrp, game_dir, tmp_dir, block_type):
    """Extract script blocks with scummrp. Returns the dump directory."""
    dump_dir = os.path.join(tmp_dir, block_type)
    os.makedirs(dump_dir, exist_ok=True)
    subprocess.run(
        [scummrp, "-g", "monkeycdalt", "-p", game_dir, "-t", block_type, "-o", "-d", dump_dir],
        capture_output=True, check=True,
    )
    return dump_dir


def find_script_files(dump_dir, prefix):
    """Find all extracted script files matching prefix."""
    results = []
    for root, dirs, files in os.walk(dump_dir):
        for f in files:
            if f.startswith(prefix):
                path = os.path.join(root, f)
                # Extract room number from path
                room = -1
                for part in path.split(os.sep):
                    if part.startswith("LFLF_"):
                        room = int(part.split("_")[1])
                results.append((path, room, f))
    return sorted(results)


def decompile(descumm, script_path):
    """Decompile a script file with descumm. Returns output text."""
    try:
        result = subprocess.run(
            [descumm, "-5", script_path],
            capture_output=True, timeout=30,
        )
        return result.stdout.decode("latin-1", errors="replace")
    except Exception:
        return ""


def extract_verb_from_obcd(obcd_path):
    """Extract the VERB chunk from an OBCD file. Returns (verb_bytes, obj_id) or (None, None)."""
    with open(obcd_path, "rb") as f:
        data = f.read()

    # Find CDHD to get object ID
    obj_id = None
    pos = data.find(b"CDHD")
    if pos >= 0:
        obj_id = struct.unpack("<H", data[pos + 8 : pos + 10])[0]

    # Find VERB chunk
    pos = data.find(b"VERB")
    if pos < 0:
        return None, obj_id

    size = struct.unpack(">I", data[pos + 4 : pos + 8])[0]
    return data[pos : pos + size], obj_id


def extract_obna_from_obcd(obcd_path):
    """Extract the OBNA name from an OBCD file. Returns (name_str, obj_id)."""
    with open(obcd_path, "rb") as f:
        data = f.read()

    obj_id = None
    pos = data.find(b"CDHD")
    if pos >= 0:
        obj_id = struct.unpack("<H", data[pos + 8 : pos + 10])[0]

    pos = data.find(b"OBNA")
    if pos < 0:
        return None, obj_id

    # Read null-terminated string after 8-byte tag+size header
    start = pos + 8
    end = data.index(b"\x00", start) if b"\x00" in data[start:] else len(data)
    name_bytes = data[start:end]
    name = "".join(chr(b) if 0x20 <= b < 0x7F else f"\\{b:03d}" for b in name_bytes)
    return name, obj_id


# Regex patterns for descumm output
RE_SET_OBJECT_NAME = re.compile(
    r'setObjectName\((\d+|VAR_ME|Local\[\d+\]|VAR_\w+)\s*,\s*"([^"]*)"'
)
RE_ACTOR_OPS_NAME = re.compile(
    r'ActorOps\((\d+|Local\[\d+\]|VAR_\w+)\s*,\s*\[.*?Name\("([^"]*)"\)'
)


def parse_decompiled(text, room, source_label, verb_obj_id=None):
    """Parse descumm output for name-setting operations. Returns list of dicts.

    verb_obj_id: if set, resolves VAR_ME to this object ID (for VERB scripts).
    """
    results = []

    for line in text.split("\n"):
        m = RE_SET_OBJECT_NAME.search(line)
        if m:
            target_raw, name = m.group(1), m.group(2)
            try:
                target_id = int(target_raw)
                target_type = "const"
            except ValueError:
                # Resolve VAR_ME in VERB scripts to the parent object ID
                if target_raw == "VAR_ME" and verb_obj_id is not None:
                    target_id = verb_obj_id
                    target_type = "const"
                else:
                    target_id = target_raw
                    target_type = "var"

            results.append({
                "kind": "object",
                "target": target_id,
                "target_type": target_type,
                "name": name,
                "room": room,
                "source": source_label,
            })

        m = RE_ACTOR_OPS_NAME.search(line)
        if m:
            target_raw, name = m.group(1), m.group(2)
            try:
                target_id = int(target_raw)
                target_type = "const"
            except ValueError:
                target_id = target_raw
                target_type = "var"

            results.append({
                "kind": "actor",
                "target": target_id,
                "target_type": target_type,
                "name": name,
                "room": room,
                "source": source_label,
            })

    return results


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <game_dir> [output_json]", file=sys.stderr)
        sys.exit(1)

    game_dir = sys.argv[1]
    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    output_path = sys.argv[2] if len(sys.argv) > 2 else os.path.join(
        repo, "translation", "monkey1", "dynamic_names.json"
    )

    scummrp = find_bin("scummrp")
    descumm_bin = find_bin("descumm")

    print("=== Extracting script blocks ===")
    tmp_dir = tempfile.mkdtemp(prefix="mi1-dynnames-")

    try:
        all_results = []

        # Extract and decompile SCRP, LSCR, ENCD, EXCD
        for block_type in ["SCRP", "LSCR", "ENCD", "EXCD"]:
            print(f"  {block_type}...", end=" ", flush=True)
            dump = extract_scripts(scummrp, game_dir, tmp_dir, block_type)
            files = find_script_files(dump, block_type if block_type != "ENCD" else "ENCD")
            if block_type in ("ENCD", "EXCD"):
                files = find_script_files(dump, block_type.split("_")[0])
                # ENCD/EXCD files are just named "ENCD" or "EXCD"
                files = []
                for root, dirs, fnames in os.walk(dump):
                    for fn in fnames:
                        if fn in (block_type, block_type.split("_")[0]):
                            fp = os.path.join(root, fn)
                            room = -1
                            for part in fp.split(os.sep):
                                if part.startswith("LFLF_"):
                                    room = int(part.split("_")[1])
                            files.append((fp, room, fn))
                files.sort()

            count = 0
            for path, room, fname in files:
                text = decompile(descumm_bin, path)
                results = parse_decompiled(text, room, f"{block_type}:{fname}")
                all_results.extend(results)
                count += len(results)
            print(f"{len(files)} files, {count} name changes")

        # Extract OBCD blocks for VERB scripts and OBNA names
        print(f"  OBCD...", end=" ", flush=True)
        obcd_dump = extract_scripts(scummrp, game_dir, tmp_dir, "OBCD")

        obna_names = {}  # obj_id -> name string
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

                # Extract OBNA name
                name, obj_id = extract_obna_from_obcd(fpath)
                if obj_id is not None and name is not None:
                    obna_names[obj_id] = name

                # Extract and decompile VERB
                verb_data, obj_id = extract_verb_from_obcd(fpath)
                if verb_data is None:
                    continue

                verb_tmp = os.path.join(tmp_dir, "verb_tmp.bin")
                with open(verb_tmp, "wb") as f:
                    f.write(verb_data)

                text = decompile(descumm_bin, verb_tmp)
                results = parse_decompiled(text, room, f"VERB:{fname}", verb_obj_id=obj_id)
                all_results.extend(results)
                verb_count += len(results)

        obcd_files = sum(1 for _, _, f in os.walk(obcd_dump) for fn in f if fn.startswith("OBCD_"))
        print(f"{obcd_files} files, {verb_count} name changes, {len(obna_names)} OBNA names")

        # Build the output structure
        print(f"\n=== Building mapping ===")

        # Group by target object (const targets only — variable targets can't be resolved statically)
        object_names = {}  # obj_id -> {obna, replacements: [{name, room, source}]}
        actor_names = {}   # actor_id -> {replacements: [{name, room, source}]}
        variable_targets = []  # unresolvable variable-targeted replacements

        for r in all_results:
            if r["target_type"] == "var":
                variable_targets.append(r)
                continue

            target_id = r["target"]

            if r["kind"] == "object":
                if target_id not in object_names:
                    obna = obna_names.get(target_id, None)
                    object_names[target_id] = {
                        "obna": obna,
                        "obna_len": len(obna) if obna else 0,
                        "has_padding": "@" in obna if obna else False,
                        "replacements": [],
                    }
                object_names[target_id]["replacements"].append({
                    "name": r["name"],
                    "len": len(r["name"]),
                    "room": r["room"],
                    "source": r["source"],
                })
            elif r["kind"] == "actor":
                if target_id not in actor_names:
                    actor_names[target_id] = {"replacements": []}
                actor_names[target_id]["replacements"].append({
                    "name": r["name"],
                    "len": len(r["name"]),
                    "room": r["room"],
                    "source": r["source"],
                })

        # Calculate max replacement length for each object
        for obj_id, data in object_names.items():
            if data["replacements"]:
                data["max_replacement_len"] = max(r["len"] for r in data["replacements"])
            else:
                data["max_replacement_len"] = 0

        output = {
            "_comment": "Dynamic name replacements extracted from MI1 SCUMM scripts. "
                        "Generated by tools/find_dynamic_names.py. "
                        "Object keys are string-encoded integers (JSON limitation).",
            "objects": {str(k): v for k, v in sorted(object_names.items())},
            "actors": {str(k): v for k, v in sorted(actor_names.items())},
            "variable_targets": variable_targets,
        }

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(output, f, indent=2, ensure_ascii=False)

        # Print summary
        padded = sum(1 for d in object_names.values() if d["has_padding"])
        with_replacements = sum(1 for d in object_names.values() if d["replacements"])
        print(f"  Objects with OBNA names: {len(obna_names)}")
        print(f"  Objects with @ padding: {padded}")
        print(f"  Objects with runtime replacements: {with_replacements}")
        print(f"  Actor name changes: {sum(len(d['replacements']) for d in actor_names.values())}")
        print(f"  Variable-target replacements (unresolvable): {len(variable_targets)}")
        print(f"\n  Written: {output_path}")

    finally:
        shutil.rmtree(tmp_dir, ignore_errors=True)


if __name__ == "__main__":
    main()
