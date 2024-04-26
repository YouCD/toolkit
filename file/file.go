package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

// Write
//
//	@Description:写入文件，会自动创建文件夹
//	@param fileData
//	@param filePath
//	@param perm
//	@return error
func Write(fileData []byte, filePath string, perm os.FileMode) error {
	base := path.Dir(filePath)
	if base != "" {
		err := os.MkdirAll(base, 0755)
		if err != nil {
			return fmt.Errorf("创建文件夹 %s failed: %w", base, err)
		}
	}
	//nolint:wrapcheck
	return os.WriteFile(filePath, fileData, perm)
}

// WriteSkipFileExist
//
//	@Description: 写入文件，如果存在则跳过
//	@param fileName
func WriteSkipFileExist(fileName string, data []byte, perm os.FileMode) error {
	_, err := os.Stat(fileName)
	// 如果文件存在 则不会有任何错误
	if err == nil {
		return nil
	}
	//   如果文件存在，还报错
	if !errors.Is(err, os.ErrNotExist) {
		return errors.WithMessagef(err, "文件存在: %s,err:%s", fileName, err.Error())
	}
	if err = os.WriteFile(fileName, data, perm); err != nil {
		return errors.WithMessagef(err, "写入文件失败: %s", fileName)
	}
	return nil
}

// WatchFS
//
//	@Description:监听文件夹,当出现target时返回true
//	@param watchSir
//	@param targetDir
//	@return bool
func WatchFS(ctx context.Context, watchSir, target string) (bool, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return false, fmt.Errorf("监听文件夹失败: %w", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Create {
					if event.Name == target {
						done <- true
					}
				}
			case <-ctx.Done():
				done <- false
				// case err, ok := <-watcher.Errors:
				//	if !ok {
				//		log.Error(err.Error())
				//		return
				//	}
			}
		}
	}()

	err = watcher.Add(watchSir)
	if err != nil {
		return false, fmt.Errorf("监听文件夹失败: %w", err)
	}

	return <-done, nil
}

// Exists
//
//	@Description: 判断所给路径文件/文件夹是否存在
//	@param path
//	@return bool
func Exists(path string) bool {
	// os.Stat获取文件信息
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}
	return true
}

// IsDir
//
//	@Description: 判断所给路径是否为文件夹
//	@param path
//	@return bool
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// DirExist
//
//	@Description:
//	判断文件夹是否存在
//	@param path
//	@return bool
func DirExist(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return fi.IsDir()
}

// IsFile
//
//	@Description: 判断所给路径是否为文件
//	@param path
//	@return bool
func IsFile(path string) bool {
	return !IsDir(path)
}

// Copy 拷贝文件
//
//	@Description:
//	@param src
//	@param dest
//	@return error
func Copy(src, dest string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件夹
	destFolder := filepath.Dir(dest)
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return fmt.Errorf("创建目标文件夹失败,src:%s,dest:%s,err: %w", src, dest, err)
	}

	stat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取源文件信息失败,src:%s,dest:%s,err: %w", src, dest, err)
	}

	// 创建目标文件 一定要加 os.O_TRUNC 这样才能覆盖
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, stat.Mode())
	if err != nil {
		return fmt.Errorf("创建目标文件失败,src:%s,dest:%s,err: %w", src, dest, err)
	}
	defer destFile.Close()

	// 使用缓冲区进行分块拷贝
	buffer := make([]byte, 8192)
	for {
		// 从源文件读取数据到缓冲区
		bytesRead, err := srcFile.Read(buffer)
		if err != nil {
			//nolint:errorlint
			if err == io.EOF {
				break // 文件读取完毕
			}
			return fmt.Errorf("读取文件失败: %w", err)
		}

		// 将缓冲区的数据写入目标文件
		if _, err = destFile.Write(buffer[:bytesRead]); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	}
	// 强制修改文件权限
	if err = destFile.Chmod(stat.Mode()); err != nil {
		return fmt.Errorf("修改文件权限失败: %w", err)
	}

	return nil
}

// CopyFolder
//
//	@Description: 拷贝文件夹,将 srcDir 下的所有文件拷贝到 dstDir文件夹下
//	@param source
//	@param destination
//	@return error
func CopyFolder(srcDir, dstDir string) error {
	// 获取源文件夹信息
	sourceInfo, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("获取源文件夹信息失败: %w", err)
	}
	var UID int
	var GID int
	if stat, ok := sourceInfo.Sys().(*syscall.Stat_t); ok {
		UID = int(stat.Uid)
		GID = int(stat.Gid)
	} else {
		UID = os.Getuid()
		GID = os.Getgid()
	}
	// 创建目标文件夹
	if err = os.MkdirAll(dstDir, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("创建目标文件夹失败: %w", err)
	}

	//  设置目标文件夹的权限和源文件夹一致
	if err = os.Chown(dstDir, UID, GID); err != nil {
		return fmt.Errorf("设置目标文件夹权限失败: %w", err)
	}

	// 遍历源文件夹中的文件和子文件夹
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("读取源文件夹信息失败: %w", err)
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(srcDir, entry.Name())
		destinationPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			// 递归拷贝子文件夹
			if err = CopyFolder(sourcePath, destinationPath); err != nil {
				return err
			}
			continue
		}

		// 拷贝文件
		if err = Copy(sourcePath, destinationPath); err != nil {
			return err
		}
	}

	return nil
}

// MoveFile
//
//	@Description: 移动文件
//	@param sourcePath
//	@param destPath
//	@return error
func MoveFile(sourcePath, destPath string) error {
	if err := Copy(sourcePath, destPath); err != nil {
		return fmt.Errorf("移动文件失败: %w", err)
	}
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("删除源文件失败: %w", err)
	}
	return nil
}

func MoveDir(sourcePath, destPath string) error {
	if err := CopyFolder(sourcePath, destPath); err != nil {
		return fmt.Errorf("移动文件夹失败: %w", err)
	}
	err := os.RemoveAll(sourcePath)
	if err != nil {
		return fmt.Errorf("删除源文件夹失败: %w", err)
	}
	return nil
}

// DirSize
//
//	@Description:文件夹大小
//	@param path
//	@return int64
//	@return error
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	if err != nil {
		return size, fmt.Errorf("计算文件夹大小失败: %w", err)
	}
	return size, nil
}

func RemoveAll(dir string) {
	_ = os.RemoveAll(dir)
}

// FromDataDir
//
//	@Description: 返回安装包中指 targetDir 目录中指定fileOrDir的绝对路径，例： 获取 release/data/a.tar 文件,或返回a.tar的绝对路径
//	@param firOrDir
//	@return absPath
func FromDataDir(targetDir, fileOrDir string) string {
	dir := path.Dir(os.Args[0])
	getwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	switch {
	case path.IsAbs(dir):
		return path.Join(dir, targetDir, fileOrDir)
	case dir == ".":
		return path.Join(getwd, targetDir, fileOrDir)
	default:
		return path.Join(getwd, dir, targetDir, fileOrDir)
	}
}

// ChownAllFile
//
//	@Description: chmod -R
//	@param dir
//	@param uid
//	@param gid
//	@return error
func ChownAllFile(dir string, uid, gid int) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil {
			return nil
		}
		return os.Chown(path, uid, gid)
	})
	if err != nil {
		return fmt.Errorf("修改权限失败: %w", err)
	}
	return nil
}

// ConvBase64Str
//
//	@Description: 将文件转换为base64字符串
//	@param file
//	@return string
func ConvBase64Str(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	return Base64EncodeByte(data)
}
