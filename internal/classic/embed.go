package classic

import _ "embed"

// scummtr binaries for each platform, embedded at compile time.
// These are populated into assets/ by scripts/build.sh before building.
// The translation file is NOT embedded — it ships as a loose file next to the
// patcher binary so users can edit it before applying.

//go:embed assets/scummtr-linux-x64
var scummtrLinux []byte

//go:embed assets/scummtr-darwin-x64
var scummtrDarwin []byte

//go:embed assets/scummtr-windows-x64.exe
var scummtrWindows []byte
