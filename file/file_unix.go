//go:build !windows
// +build !windows

package file

import (
	"os"
	"syscall"
)

func getFileUIDGID(info os.FileInfo) (int, int) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid), int(stat.Gid)
	}
	return os.Getuid(), os.Getgid()
}
