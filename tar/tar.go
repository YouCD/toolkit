package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
	"github.com/youcd/toolkit/file"
)

type Progress struct {
	FileCount int
	DirCount  int
}

// ExtractTarZstOrGzipFile
//
//	@Description: 解压tar.zst或者tar.gz
//	@param src
//	@param dest
//	@param progressChan
//	@return error
//
//nolint:gocognit
func ExtractTarZstOrGzipFile(src, dest string, progressChan chan string) error {
	fileHandle, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开文件: %s,err：%w", src, err)
	}
	defer fileHandle.Close()

	var pg Progress
	var reader io.Reader

	switch strings.ToLower(filepath.Ext(src)) {
	case ".gz":
		gzipReader, err := gzip.NewReader(fileHandle)
		if err != nil {
			return fmt.Errorf("读取文件: %s,err：%w", src, err)
		}
		reader = gzipReader
		defer gzipReader.Close()
	case ".zst":
		zstReader, err := zstd.NewReader(fileHandle)
		if err != nil {
			return fmt.Errorf("读取文件: %s,err：%w", src, err)
		}
		reader = zstReader
		defer zstReader.Close()
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取文件: %s,err：%w", src, err)
		}

		target := filepath.Join(dest, filepath.Clean(header.Name))

		switch header.Typeflag {
		// 文件夹
		case tar.TypeDir:
			pg.DirCount++
			pushMsg(pg, progressChan)
			if err := os.MkdirAll(target,  os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("创建文件夹失败: %s,err：%w", target, err)
			}
			// 强制设置权限（覆盖原有的）
			err = os.Chmod(target, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if err := os.Chown(target, header.Uid, header.Gid); err != nil {
				return fmt.Errorf("修改文件所有者失败: %s,err：%w", target, err)
			}

		// 普通文件
		case tar.TypeReg:
			pg.FileCount++
			pushMsg(pg, progressChan)
			writeFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("创建文件失败: %s,err：%w", target, err)
			}

			// 强制修改文件权限
			if err = writeFile.Chmod(os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("修改文件权限失败: %s,err：%w", target, err)
			}

			if err := os.Chown(target, header.Uid, header.Gid); err != nil {
				return fmt.Errorf("修改文件所有者失败: %s,err：%w", target, err)
			}
			//nolint:gosec
			if _, err := io.Copy(writeFile, tarReader); err != nil {
				return fmt.Errorf("写入文件失败: %s,err：%w", target, err)
			}
			writeFile.Close()
		// 其他类型 没有处理
		default:
		}
	}
	return nil
}
func pushMsg(progress Progress, progressChan chan string) {
	if progressChan == nil {
		return
	}
	progressChan <- fmt.Sprintf("file:%d dir:%d", progress.FileCount, progress.DirCount)
}

// Dir
//
//	@Description: 压缩指定文件
//	@param src 源目录
//	@param dst  目标文件
//	@return err
func Dir(src, dst string) error {
	data, err := tarDir(src)
	if err != nil {
		return err
	}

	fileHandle, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("打开文件失败: %s,err：%w", dst, err)
	}

	defer fileHandle.Close()

	if _, err = fileHandle.Write(data); err != nil {
		return fmt.Errorf("写入文件失败: %s,err：%w", dst, err)
	}
	return nil
}

// GzOrZstFileWithDirFunc
//
//	@Description: 打包压缩 tar.gz或者tar.zst
//	@param src
//	@param dst
//	@param dirFunc
//	@return err
func GzOrZstFileWithDirFunc(src, dst string, dirFunc dirFunc) error {
	// 文件已存在则跳过
	if file.Exists(dst) {
		return nil
	}
	// 创建文件
	fileHandle, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %s,err：%w", dst, err)
	}
	defer fileHandle.Close()

	// 添加 gzip 压缩，
	var writer io.Writer
	switch strings.ToLower(filepath.Ext(dst)) {
	case ".gz":
		gzipWriter := gzip.NewWriter(fileHandle)
		writer = gzipWriter
		defer gzipWriter.Close()
	case ".zst":
		zstReader, err := zstd.NewWriter(fileHandle)
		if err != nil {
			return fmt.Errorf("读取文件: %s,err：%w", src, err)
		}
		writer = zstReader
		defer zstReader.Close()
	}

	// 创建 tar 包
	tw := tar.NewWriter(writer)
	defer tw.Close()
	err = tarDirHandler(src, dirFunc, tw)
	if err != nil {
		return fmt.Errorf("打包失败: %s,err：%w", dst, err)
	}

	return nil
}

// DirRow
//
//	@Description:  压缩指定文件夹
//	@param src
//	@return data 压缩后的数据
//	@return err
func DirRow(src string) ([]byte, error) {
	return tarDir(src)
}

// tarDir
//
//	@Description: 压缩目录
//	@param src
//	@return []byte
//	@return error
func tarDir(src string) ([]byte, error) {
	// 创建文件
	data := &bytes.Buffer{}

	// 添加 gzip 压缩，
	gw := gzip.NewWriter(data)
	defer gw.Close()
	// 创建 tar 包
	tw := tar.NewWriter(gw)
	defer tw.Close()
	err := tarDirHandler(src, nil, tw)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

// dirFunc 自定义
// @Description: 压缩时自定义目录结构 例如: /tmp/etc/a.txt -> /etc/a.txt
type dirFunc func(fileName string) string

func DirRowWithDirFunc(src string, dirFunc dirFunc) ([]byte, error) {
	return tarDirWithFunc(src, dirFunc)
}

// tarDirWithFunc
//
//	@Description: 压缩时自定义目录结构 例如: /tmp/etc/a.txt -> /etc/a.txt
//	@param src
//	@return []byte
//	@return error
//

func tarDirWithFunc(src string, dirFunc dirFunc) ([]byte, error) {
	// 创建文件
	data := &bytes.Buffer{}

	// 添加 gzip 压缩，
	gw := gzip.NewWriter(data)
	// 创建 tar 包
	tw := tar.NewWriter(gw)

	err := tarDirHandler(src, dirFunc, tw)
	if err != nil {
		return nil, err
	}
	// 确保 tar 包中的数据被刷新到 gzip 压缩流中
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 tar 包: %w", err)
	}

	// 确保 gzip 压缩流中的数据被刷新到 data 中
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 gzip 压缩流: %w", err)
	}

	return data.Bytes(), nil
}

//nolint:wrapcheck
func tarDirHandler(src string, dirFunc dirFunc, tw *tar.Writer) error {
	return filepath.Walk(src, func(fileName string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("递归文件夹%s,err: %w", src, err)
		}
		hdr, filepathErr := tar.FileInfoHeader(fi, "")
		if filepathErr != nil {
			return fmt.Errorf("文件头信息: %w", filepathErr)
		}
		if dirFunc != nil {
			// 自定义文件夹路径
			hdr.Name = dirFunc(fileName)
		} else {
			hdr.Name = fileName
		}

		// 写入文件信息
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("写入文件信息: %w", err)
		}

		// 判断下文件是否是标准文件
		if !fi.Mode().IsRegular() {
			return nil
		}
		// 打开文件
		fr, openErr := os.Open(fileName)
		if openErr != nil {
			return fmt.Errorf("打开文件: %w", openErr)
		}
		defer func() {
			_ = fr.Close()
		}()
		// copy 文件数据到 tw
		_, err = io.Copy(tw, fr)
		if err != nil {
			return fmt.Errorf("拷贝文件: %w", err)
		}

		return nil
	})
}

func FileWithFunc(fileName string, dirFunc dirFunc) ([]byte, error) {
	// 创建文件
	data := &bytes.Buffer{}

	// 添加 gzip 压缩，
	gw := gzip.NewWriter(data)

	// 创建 tar 包
	tw := tar.NewWriter(gw)

	// 打开文件
	fr, openErr := os.Open(fileName)
	if openErr != nil {
		return nil, fmt.Errorf("打开文件: %w", openErr)
	}
	info, err := fr.Stat()
	if err != nil {
		return nil, fmt.Errorf("文件信息: %w", err)
	}
	hdr, filepathErr := tar.FileInfoHeader(info, "")
	if filepathErr != nil {
		return nil, fmt.Errorf("文件头信息: %w", filepathErr)
	}

	hdr.Name = fileName
	if dirFunc != nil {
		// 自定义文件夹路径
		hdr.Name = dirFunc(fileName)
	}

	// 写入文件信息
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, errors.WithMessage(err, "写入文件信息")
	}

	// copy 文件数据到 tw
	_, err = io.Copy(tw, fr)
	if err != nil {
		return nil, errors.WithMessage(err, "复制文件数据")
	}

	// 确保 tar 包中的数据被刷新到 gzip 压缩流中
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("关闭tar文件: %w", err)
	}

	// 确保 gzip 压缩流中的数据被刷新到 data 中
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 gzip 压缩流: %w", err)
	}
	if err := fr.Close(); err != nil {
		return nil, fmt.Errorf("关闭文件: %w", err)
	}

	return data.Bytes(), nil
}
