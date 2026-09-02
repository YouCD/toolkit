package file

import (
	"os"
	"path/filepath"
)

// ChmodRecursive 递归修改文件权限
func ChmodRecursive(path string, mode os.FileMode) error {
	return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(p, mode)
	})
}
