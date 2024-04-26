package tar

import (
	"os"
	"testing"
)

func TestTarDirWithDirFunc(t *testing.T) {
	if err := GzOrZstFileWithDirFunc("/home/ycd/fireCloud/source_code/ssc_cd/data", "/tmp/os_repo.tar.zst", nil); err != nil {
		t.Error(err)
	}
}

func TestDirRowWithDirFunc(t *testing.T) {
	bytes, err := DirRowWithDirFunc("/home/ycd/fireCloud/source_code/ssc_cd/data", nil)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile("/tmp/os_repo.tar.gz", bytes, 0644)
	if err != nil {
		t.Error(err)
	}
}

func TestFileWithFunc(t *testing.T) {
	sysinfoData, err := FileWithFunc("/home/ycd/self_data/source_code/go-source/shuttle/internal/shuttle/stage/check/sysinfo", nil)
	err = os.WriteFile("/tmp/sysinfo.tar.gz", sysinfoData, 0644)
	if err != nil {
		t.Error(err)
	}
}
