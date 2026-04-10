//go:build !buildpatcher

package charset

// Stub variables — populated by charset_embed.go when built with -tags buildpatcher.
// Calling Patch or PatchVerbLayout without the buildpatcher tag will panic.
var (
	patchedChar0001 []byte
	patchedChar0002 []byte
	patchedChar0003 []byte
	patchedChar0004 []byte
	patchedChar0006 []byte

	scummrpLinux   []byte
	scummrpDarwin  []byte
	scummrpWindows []byte
)
