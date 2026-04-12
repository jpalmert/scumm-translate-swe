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

**Binary patch applied** (`patchAutosaveTimer` in `cmd/patcher/se.go`): raises the threshold from
`300.0` to `9999999.0` at file offset `0xed010` in `MISE.exe`, effectively disabling autosave.

**Why autosave crashes with modified MONKEY1.001:** The save function (`FUN_0049a610`) walks
resource tables and serialises them using internal byte offsets. When scummtr's raw mode (`-r`)
re-encodes LFLF blocks, it changes the byte layout and adjusts all offsets within MONKEY1.001.
However, the SE engine holds those offsets in memory from the initial file load. On autosave the
engine writes the original (pre-modification) offsets to disk; when restoring from such a save
file the offsets point into the wrong locations, causing a crash.
