//go:build buildpatcher

package charset

import (
	"encoding/binary"
	"testing"
)

// ASSET-001..005: Embedded CHAR assets are valid CHAR blocks.
func TestPatchedCharAssets(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"CHAR_0001", patchedChar0001},
		{"CHAR_0002", patchedChar0002},
		{"CHAR_0003", patchedChar0003},
		{"CHAR_0004", patchedChar0004},
		{"CHAR_0006", patchedChar0006},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.data) < 8 {
				t.Fatalf("%s too short: %d bytes", tc.name, len(tc.data))
			}
			if string(tc.data[0:4]) != "CHAR" {
				t.Errorf("%s tag = %q, want CHAR", tc.name, tc.data[0:4])
			}
			size := int(binary.BigEndian.Uint32(tc.data[4:]))
			if size != len(tc.data) {
				t.Errorf("%s size field = %d, actual = %d", tc.name, size, len(tc.data))
			}
		})
	}
}

// ASSET-007: Embedded scummrp binaries are non-empty.
func TestScummrpBinariesEmbedded(t *testing.T) {
	bins := map[string][]byte{
		"linux":   scummrpLinux,
		"darwin":  scummrpDarwin,
		"windows": scummrpWindows,
	}
	for name, bin := range bins {
		if len(bin) == 0 {
			t.Errorf("scummrp %s binary is empty", name)
		}
	}
}
