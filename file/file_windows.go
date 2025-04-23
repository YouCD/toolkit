//go:build windows
// +build windows

package file

import (
	"os"
)

func getFileUIDGID(info os.FileInfo) (int, int) {
	// Windows doesn't support Unix-style UID/GID
	return -1, -1
}
