package edit

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
)

const DefaultEditor = "vi"

type configData struct {
	data     []byte
	fileInfo fs.FileInfo
}
type ConfigEdit struct {
	configData configData
	filePath   string
}

func NewConfigEdit(filePath string) (*ConfigEdit, error) {
	c := new(ConfigEdit)
	read, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件出错:%w", err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息出错:%w", err)
	}

	c.configData = configData{
		data:     read,
		fileInfo: fileInfo,
	}
	c.filePath = filePath
	return c, nil
}

// openFileInEditor
//
//	@Description: 调用系统编辑器打开文件
//	@receiver c
//	@param filename
//	@return error
func (c ConfigEdit) openFileInEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = DefaultEditor
	}

	// 判断外部命令是否存在
	executable, err := exec.LookPath(editor)
	if err != nil {
		return fmt.Errorf("editor not found:%w", err)
	}

	cmd := exec.Command(executable, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	//nolint:wrapcheck
	return cmd.Run()
}

type verifyFunc func(filename string) error

// EditConfig
//
//	@Description: 编辑文件
//	@receiver c
//	@param data
//	@return error
func (c ConfigEdit) EditConfig(verify verifyFunc) error {
	// 1. 创建临时文件
	file, err := os.CreateTemp(os.TempDir(), "*.yaml")
	if err != nil {
		return fmt.Errorf("CreateTemp() err:%w", err)
	}
	name := file.Name()

	// 2. 将初始化数据写入临时文件中
	if _, err = io.Copy(file, bytes.NewReader(c.configData.data)); err != nil {
		return fmt.Errorf("写入临时文件 err:%w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("关闭临时文件 err:%w", err)
	}

	// 3. 调用系统命令编辑已经有数据的文件
	if err = c.openFileInEditor(name); err != nil {
		return fmt.Errorf("编辑文件 err:%w", err)
	}

	// 4. 校验 配置文件
	if verify != nil {
		if err = verify(name); err != nil {
			return fmt.Errorf("配置文件格式错误:%w", err)
		}
	}

	// 5.再次打开编辑好的文件
	byteData, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("读取配置文件出错:%w", err)
	}

	// 6. 校验成功写入新的配置内容
	err = os.WriteFile(c.filePath, byteData, c.configData.fileInfo.Mode())
	if err != nil {
		return fmt.Errorf("写入配置文件出错:%w", err)
	}

	return nil
}
