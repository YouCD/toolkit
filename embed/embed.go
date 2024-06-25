package embed

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Embed struct {
	EmbedData embed.FS
}

func NewEmbed(embedData embed.FS) *Embed {
	return &Embed{EmbedData: embedData}
}

// EmbedDataMetaInfo
//
//	@Description: EmbedDataMetaInfo 获取文件信息
//	@param srcFile
//	@return error
//	@return fs.FileInfo
//	@return error
func (e *Embed) EmbedDataMetaInfo(filename string) (fs.FileInfo, error) {
	file, err := e.EmbedData.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("打开源文件失败: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取源文件信息失败: %w", err)
	}
	return stat, nil
}

// CopyEmbedDataFolderWithSkipFileFunc
//
//	@Description: 跳过文件
//	@param embedSrcDir
//	@param dstDir
//	@param skipFileFunc
//	@return error
func (e *Embed) CopyEmbedDataFolderWithSkipFileFunc(embedSrcDir, dstDir string, skipFileFunc func(filename string) bool) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目标文件夹失败: %w", err)
	}
	// 获取源文件夹信息
	stat, err := e.EmbedDataMetaInfo(embedSrcDir)
	if err != nil {
		return err
	}

	// 创建目标文件夹
	if err = os.MkdirAll(dstDir, stat.Mode()); err != nil {
		return fmt.Errorf("创建目标文件夹失败: %w", err)
	}

	// 遍历源文件夹中的文件和子文件夹
	entries, err := e.EmbedData.ReadDir(embedSrcDir)
	if err != nil {
		return fmt.Errorf("读取源文件夹信息失败: %w", err)
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(embedSrcDir, entry.Name())
		if skipFileFunc != nil {
			if skipFileFunc(sourcePath) {
				continue
			}
		}
		destinationPath := filepath.Join(dstDir, entry.Name())
		// 递归拷贝子文件夹
		if entry.IsDir() {
			if err = e.CopyEmbedDataFolderWithSkipFileFunc(sourcePath, destinationPath, skipFileFunc); err != nil {
				return err
			}
			continue
		}
		// 拷贝文件
		if err = e.copyEmbedDataFile(sourcePath, destinationPath); err != nil {
			return err
		}
	}
	return nil
}

// copyEmbedDataFile
//
//	@Description: 从embed文件夹中复制文件
//	@param embedSrcFile
//	@param dstFile
//	@return error
func (e *Embed) copyEmbedDataFile(embedSrcFile, dstFile string) error {
	stat, err := e.EmbedDataMetaInfo(embedSrcFile)
	if err != nil {
		return err
	}

	fileData, err := e.EmbedData.ReadFile(embedSrcFile)
	if err != nil {
		return fmt.Errorf("读取源文件失败: %w", err)
	}

	// 写入目标文件
	if err = os.WriteFile(dstFile, fileData, stat.Mode()); err != nil {
		return fmt.Errorf("写入目标文件失败: %w", err)
	}
	return nil
}

// CopyEmbedDataFile
//
//	@Description: 从embed文件夹中复制文件
//	@receiver e
//	@param embedSrcFile
//	@param dstFile
//	@return error
func (e *Embed) CopyEmbedDataFile(embedSrcFile, dstFile string) error {
	if err := os.MkdirAll(filepath.Dir(dstFile), 0755); err != nil {
		return fmt.Errorf("创建目标文件夹失败: %w", err)
	}
	// 拷贝文件
	return e.copyEmbedDataFile(embedSrcFile, dstFile)
}
