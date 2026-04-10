package pak_test

import (
    "os"
    "strings"
    "testing"
    "scumm-patcher/internal/pak"
)

func TestExtractMisc(t *testing.T) {
    _, _, _, entries, err := pak.Read("../../game/monkey1/Monkey1.pak")
    if err != nil {
        t.Skip(err)
    }
    for _, e := range entries {
        lower := strings.ToLower(e.Name)
        if strings.HasSuffix(lower, ".xml") || strings.HasSuffix(lower, ".csv") || lower == "tweaks.txt" {
            out := strings.ReplaceAll(e.Name, "/", "_")
            os.WriteFile("/tmp/claude/"+out, e.Data, 0644)
            t.Logf("Extracted: %s (%d bytes)", e.Name, len(e.Data))
        }
    }
}
