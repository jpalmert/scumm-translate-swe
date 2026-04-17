package charset

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// scummrpEnv holds the paths and runner function for a temporary scummrp
// working environment. The caller must call cleanup() when done.
type scummrpEnv struct {
	tmpDir  string
	dumpDir string
	run     func(args ...string) error
	cleanup func()
}

// setupScummrp selects the platform-appropriate scummrp binary, writes it to a
// temp directory, and returns a ready-to-use scummrpEnv. The caller must call
// env.cleanup() (typically via defer) to remove the temp directory.
func setupScummrp(tmpPrefix string) (*scummrpEnv, error) {
	var scummrpBin []byte
	var scummrpName string
	switch runtime.GOOS {
	case "linux":
		scummrpBin = scummrpLinux
		scummrpName = "scummrp"
	case "darwin":
		scummrpBin = scummrpDarwin
		scummrpName = "scummrp"
	case "windows":
		scummrpBin = scummrpWindows
		scummrpName = "scummrp.exe"
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	tmpDir, err := os.MkdirTemp("", tmpPrefix)
	if err != nil {
		return nil, err
	}

	scummrpPath := filepath.Join(tmpDir, scummrpName)
	if err := os.WriteFile(scummrpPath, scummrpBin, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	return &scummrpEnv{
		tmpDir:  tmpDir,
		dumpDir: filepath.Join(tmpDir, "dump"),
		run: func(args ...string) error {
			cmd := exec.Command(scummrpPath, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		cleanup: func() { os.RemoveAll(tmpDir) },
	}, nil
}
