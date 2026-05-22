package doctor

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func writableForExisting(path string, mode os.FileMode) bool {
	if mode.IsDir() {
		return false
	}
	return unix.Access(path, unix.W_OK) == nil
}

func writableForMissing(path string) bool {
	dir := filepath.Clean(filepath.Dir(path))
	for {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return false
			}
			return unix.Access(dir, unix.W_OK|unix.X_OK) == nil
		}
		if !os.IsNotExist(err) {
			return false
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}
