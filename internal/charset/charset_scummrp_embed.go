package charset

import _ "embed"

// scummrp binaries for each supported platform, embedded at compile time.
// These are committed to git under assets/ and always available.

//go:embed assets/scummrp-linux-x64
var scummrpLinux []byte

//go:embed assets/scummrp-darwin-x64
var scummrpDarwin []byte

//go:embed assets/scummrp-windows-x64.exe
var scummrpWindows []byte
