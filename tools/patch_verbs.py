#!/usr/bin/env python3
"""
patch_verbs.py — Patch verb button screen positions in a SCUMM v5 SCRP_0022 block.

In Monkey Island 1 (SCUMM v5), each verb button carries both its action ID and its
screen coordinates inside the SCRP_0022 script block. The coordinates determine
where the button appears in the 3x3 grid on screen.

Original English layout:

    Left (x=0x16) | Middle (x=0x48) | Right (x=0x7C)
    Give          | Pick up          | Use             <- Top    (y=0x9B)
    Open          | Look at          | Push            <- Middle (y=0xAB)
    Close         | Talk to          | Pull            <- Bottom (y=0xBB)

Patched layout (Swedish — shorter words right, longer words left/middle):

    Left (x=0x16) | Middle (x=0x48) | Right (x=0x7C)
    Open/Öppna    | Look at/Titta   | Give/Ge         <- Top    (y=0x9B)
    Close/Stäng   | Talk to/Tala    | Pick up/Ta      <- Middle (y=0xAB)
    Push/Putta    | Use/Använd      | Pull/Dra        <- Bottom (y=0xBB)

Binary structure of each verb entry in SCRP_0022:
    [func_code] 0x09 0x02 [ASCII label...] 0x00 0x13 0x12 [shortcut] 0x05 [X] 0x00 [Y] ...

Usage:
    python3 tools/patch_verbs.py <input_scrp_0022> <output_scrp_0022>

The input must be the raw SCRP_0022 binary as dumped by scummrp (XOR-decoded, with
block header). The output is written to the specified path.
"""

import sys
import os


# Verb table: (func_code, description, new_x, new_y)
# func_code uniquely identifies each verb action in the SCRP_0022 binary.
VERBS = [
    (0x04, "Give",    0x7C, 0x9B),  # Left/Top     → Right/Top
    (0x02, "Open",    0x16, 0x9B),  # Left/Middle  → Left/Top
    (0x03, "Close",   0x16, 0xAB),  # Left/Bottom  → Left/Middle
    (0x09, "Pick up", 0x7C, 0xAB),  # Mid/Top      → Right/Middle
    (0x08, "Look at", 0x48, 0x9B),  # Mid/Middle   → Mid/Top
    (0x0A, "Talk to", 0x48, 0xAB),  # Mid/Bottom   → Mid/Middle
    (0x07, "Use",     0x48, 0xBB),  # Right/Top    → Mid/Bottom
    (0x05, "Push",    0x16, 0xBB),  # Right/Middle → Left/Bottom
    (0x06, "Pull",    0x7C, 0xBB),  # Right/Bottom → unchanged
]


def find_verb_x_offset(data: bytes, func_code: int, description: str) -> int:
    """
    Locate the X coordinate byte for a verb entry in SCRP_0022.

    Searches for the pattern:
        func_code 0x09 0x02 <printable ASCII label> 0x00 0x13 0x12 <shortcut> 0x05

    Returns the byte offset of X (Y is at offset + 2).
    Raises ValueError if the pattern is not found or matches more than once.
    """
    candidates = []
    i = 0
    while i < len(data) - 8:
        if data[i] == func_code and data[i + 1] == 0x09 and data[i + 2] == 0x02:
            # Verify the label is printable ASCII (ends at a null byte).
            j = i + 3
            while j < len(data) and data[j] != 0x00:
                if data[j] < 0x20 or data[j] > 0x7E:
                    break
                j += 1
            if j >= len(data) or data[j] != 0x00:
                i += 1
                continue
            # j is at null terminator; expect 0x13 0x12 <shortcut> 0x05 next.
            if j + 4 < len(data) and data[j + 1] == 0x13 and data[j + 2] == 0x12 and data[j + 4] == 0x05:
                candidates.append(j + 5)  # offset of X byte
        i += 1

    if not candidates:
        raise ValueError(
            f"Verb {description!r} (func_code=0x{func_code:02X}) not found in SCRP_0022"
        )
    if len(candidates) > 1:
        raise ValueError(
            f"Verb {description!r} (func_code=0x{func_code:02X}) matched {len(candidates)} "
            f"times — ambiguous; check the SCRP_0022 format"
        )
    return candidates[0]


def patch(input_path: str, output_path: str) -> None:
    with open(input_path, "rb") as f:
        data = bytearray(f.read())

    print(f"Patching verb positions in {input_path} ({len(data)} bytes)")
    for func_code, description, new_x, new_y in VERBS:
        x_off = find_verb_x_offset(bytes(data), func_code, description)
        old_x, old_y = data[x_off], data[x_off + 2]
        data[x_off] = new_x
        data[x_off + 2] = new_y
        moved = "→" if (old_x, old_y) != (new_x, new_y) else "  (unchanged)"
        print(
            f"  {description:<8} (0x{func_code:02X}): "
            f"x=0x{old_x:02X},y=0x{old_y:02X} → x=0x{new_x:02X},y=0x{new_y:02X}{moved if isinstance(moved, str) and moved.startswith('  ') else ''}"
        )

    os.makedirs(os.path.dirname(os.path.abspath(output_path)), exist_ok=True)
    with open(output_path, "wb") as f:
        f.write(data)
    print(f"Written: {output_path} ({len(data)} bytes)")


def main() -> None:
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <input_scrp_0022> <output_scrp_0022>", file=sys.stderr)
        sys.exit(1)
    patch(sys.argv[1], sys.argv[2])


if __name__ == "__main__":
    main()
