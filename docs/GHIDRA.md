# Ghidra — Reverse Engineering Reference

Ghidra is a free, open-source reverse engineering suite from the NSA. It disassembles and
decompiles binaries to C pseudocode, making it useful for analysing game engine binaries like
`MISE.exe` without needing source code.

## Installation

**Version installed:** 12.0.4 (`/home/jpalmert/tools/ghidra_12.0.4_PUBLIC`)

**Java requirement:** Java 17 or later. Java 21 is available at:
```
/home/jpalmert/.windsurf/extensions/redhat.java-1.50.0-linux-x64/jre/21.0.9-linux-x86_64/bin/java
```

**Platform:** Ghidra is Java-based — a single download runs on Linux, macOS, and Windows. There
are no separate platform binaries.

**Download:** GitHub releases at `NationalSecurityAgency/ghidra`.

## Headless analysis

Use `analyzeHeadless` (in `support/`) for scripted, non-interactive analysis:

```bash
JAVA_HOME=/home/jpalmert/.windsurf/extensions/redhat.java-1.50.0-linux-x64/jre/21.0.9-linux-x86_64

# First-time import + analysis (slow — ~50s for MISE.exe):
$JAVA_HOME/../.. /home/jpalmert/tools/ghidra_12.0.4_PUBLIC/support/analyzeHeadless \
    /home/jpalmert/ghidra_projects mise_project \
    -import /path/to/MISE.exe \
    -processor x86:LE:32:default \
    -analysisTimeoutPerFile 180

# Re-run a script on an already-analysed project (fast):
JAVA_HOME=... /home/jpalmert/tools/ghidra_12.0.4_PUBLIC/support/analyzeHeadless \
    /home/jpalmert/ghidra_projects mise_project \
    -process MISE.exe \
    -noanalysis \
    -scriptPath /home/jpalmert/ghidra_scripts \
    -postScript MyScript.java
```

**Important:** use `-scriptPath` pointing to a dedicated directory. If you point at `/tmp`,
Ghidra will attempt to compile all `.java` files there, which fails when unrelated Java files
are present.

Scripts live in `/home/jpalmert/ghidra_scripts/`.

## Writing scripts

Ghidra headless scripts must be Java files (Python/Jython requires PyGhidra setup). Minimal
template:

```java
// Short description
//@category Analysis

import ghidra.app.decompiler.*;
import ghidra.app.script.GhidraScript;
import ghidra.program.model.address.Address;
import ghidra.program.model.listing.Function;
import ghidra.util.task.ConsoleTaskMonitor;

public class MyScript extends GhidraScript {
    @Override
    public void run() throws Exception {
        DecompInterface decompiler = new DecompInterface();
        decompiler.openProgram(currentProgram);

        Address addr = currentProgram.getAddressFactory().getAddress("0x41d1f0");
        Function func = currentProgram.getFunctionManager().getFunctionAt(addr);

        if (func != null) {
            DecompileResults result = decompiler.decompileFunction(func, 60, new ConsoleTaskMonitor());
            if (result.decompileCompleted()) {
                println(result.getDecompiledFunction().getC());
            }
        }
        decompiler.dispose();
    }
}
```

## MISE.exe findings

The analysed project is at `/home/jpalmert/ghidra_projects/mise_project`.

Key locations identified via disassembly and Ghidra decompilation:

| Address (VA) | Description |
|---|---|
| `0x41d1f0` | Autosave tick function — accumulates game time and fires autosave |
| `0x4ed010` | Autosave threshold constant — IEEE 754 double `300.0` (5 minutes) |
| `0x49a610` | Save function — formats `savegame.%03d`, serialises resource tables |
| `0x49ad30` | Load function — opens savegame file, calls `FUN_0049b3e0` then resource loop |
| `0x49b3e0` | State serialiser — fixed-size + variable-size state blocks; includes version check |
| `0x49ab60` | Resource-data restore loop body — allocates heap + reads resource bytes from save |
| `0x4990e0` | Resource heap allocator — malloc with LRU eviction; uses ESI as requested size |
| `0x4a8170` | Custom heap malloc — linked free-list allocator (`DAT_005c07a8` = free-list head) |

**Autosave logic** (from decompiled `FUN_0041d1f0`):
```c
// Accumulate elapsed game time into this->0x48
in_XMM0_Qa = *(double *)((int)this + 0x48) + local_18.QuadPart;
*(double *)((int)this + 0x48) = in_XMM0_Qa;
// ...
if (_DAT_004ed010 < in_XMM0_Qa) {   // _DAT_004ed010 = 300.0
    FUN_0044eea0(1);                 // triggers autosave
}
```

**Binary patch 1** (`patchAutosaveTimer` in `cmd/patcher/se.go`): raises the threshold from
`300.0` to `9999999.0` at file offset `0xed010` in `MISE.exe`, effectively disabling autosave.

**Binary patch 2 — save/load crash fix** (`patchSaveLoadCrash` in `cmd/patcher/se.go`):

**Root cause:** `FUN_0049ad30` (load function) calls `FUN_0049b260` to free most heap resources,
then runs a loop calling `FUN_0049ab60` to restore each saved resource block from the save file.
`FUN_0049ab60` calls `FUN_004990e0` to allocate a heap buffer, then reads the saved resource bytes
into it. The SCUMM heap (`DAT_005c07a8`) is a fixed-size linked free-list. Resource types 0xc,
2, and 10 are NOT freed by `FUN_0049b260` — they remain locked in the heap.

After scummtr raw-mode injection, LFLF blocks are larger (Swedish strings are longer). The locked
resources consume the same heap space as before, but the saved resource blocks are bigger. When
the total saved-resource size exceeds available heap (fixed heap minus locked resources), `FUN_004990e0`
returns 0. `FUN_00499050` then writes to address `0 + 4 = 4` → access violation crash.

This happens even for a fresh save on the patched game: the heap was sized for original resources;
the patched resources exceed available capacity during the simultaneous restore.

**Fix:** patch the first 3 bytes of `FUN_0049ab60` (file offset `0x9ab60`) to `31 C0 C3`
(`XOR EAX,EAX; RET`), making the resource-data loop exit immediately. Resources are then loaded
from the patched MONKEY1.001 on demand (via `FUN_00496780`). Script PCs are preserved because
`FUN_0049b3e0`'s variable-size state block (`FUN_0049a5c0(DAT_005c4460, _DAT_005c259c * 4)`)
covers `DAT_005c4680` — the per-script PC array — so PCs are correctly restored without the loop.
