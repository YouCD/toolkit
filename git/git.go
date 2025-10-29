package git

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	cryptoSSH "golang.org/x/crypto/ssh"
)

type Git struct {
	sshURL     string
	storage    *memory.Storage
	publicKeys *ssh.PublicKeys
	ref        string
	fs         billy.Filesystem
	Repository *git.Repository
}

// NewGit
//
//	@Description: 使用的是内存临时存储
//	@param sshURLOrHTTPURL
//	@param ref
//	@return *Git
//	@return error
func NewGit(sshURLOrHTTPURL, ref string) (*Git, error) {
	sshURL := sshURLOrHTTPURL
	parse, err := url.Parse(sshURLOrHTTPURL)
	if err == nil {
		sshURL = fmt.Sprintf("git@%s:%s", parse.Host, parse.Path)
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("user home dir error: %w", err)
	}

	pemFile := filepath.Join(dir, ".ssh", "id_rsa")
	publicKeys, err := ssh.NewPublicKeysFromFile("git", pemFile, "")
	if err != nil {
		return nil, fmt.Errorf("ssh key error: %w", err)
	}
	//nolint:gosec
	publicKeys.HostKeyCallback = cryptoSSH.InsecureIgnoreHostKey()

	return &Git{
		sshURL:     sshURL,
		storage:    memory.NewStorage(),
		publicKeys: publicKeys,
		ref:        ref,
		fs:         memfs.New(),
	}, nil
}

// NewGitInit
//
//	@Description: 初始化git，直接初始化
//	@param sshURLOrHttpURL
//	@param ref
//	@return *Git
//	@return error
func NewGitInit(sshURLOrHTTPURL, ref string) (*Git, error) {
	g, err := NewGit(sshURLOrHTTPURL, ref)
	if err != nil {
		return nil, err
	}
	err = g.Clone(1)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// PlainClone
//
//	@Description:
//	@receiver g
//	@param path clone到指定的路径
//	@param depth
//	@return error
func (g *Git) PlainClone(path string, depth int) error {
	repository, err := git.PlainClone(path, false, &git.CloneOptions{
		InsecureSkipTLS: true,
		URL:             g.sshURL,
		ReferenceName:   plumbing.ReferenceName(g.ref),
		Auth:            g.publicKeys,
		Depth:           depth,
	})
	if err != nil {
		return fmt.Errorf("git clone error: %w", err)
	}
	g.Repository = repository
	return nil
}

// Clone
//
//	@Description: 克隆代码
//	@receiver g
func (g *Git) Clone(depth int) error {
	Repository, err := git.Clone(g.storage, g.fs, &git.CloneOptions{
		InsecureSkipTLS: true,
		URL:             g.sshURL,
		ReferenceName:   plumbing.ReferenceName(g.ref),
		Auth:            g.publicKeys,
		Depth:           depth,
	})
	if err != nil {
		return fmt.Errorf("git clone error: %w", err)
	}
	g.Repository = Repository
	return nil
}

// ReadFile
//
//	@Description: 读取文件
//	@receiver g
//	@param file
//	@return []byte
//	@return error
func (g *Git) ReadFile(file string) ([]byte, error) {
	openFile, err := g.fs.OpenFile(file, os.O_RDONLY, 0)
	defer func() {
		if openFile != nil {
			_ = openFile.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("g.fs.OpenFile(), err:%w", err)
	}
	//nolint:wrapcheck
	return io.ReadAll(openFile)
}

// ReadDirOrFile
//
//	@Description: 读取目录或者文件
//	@receiver g
//	@param dirOrFile
//	@param data
//	@return error
func (g *Git) ReadDirOrFile(dirOrFile string, data map[string][]byte) error {
	openTarget, err := g.fs.Stat(dirOrFile)
	if err != nil {
		return fmt.Errorf("g.fs.Stat(), err:%w", err)
	}

	if openTarget.IsDir() {
		dir, err := g.fs.ReadDir(dirOrFile)
		if err != nil {
			return fmt.Errorf("g.fs.ReadDir(), err:%w", err)
		}
		for _, info := range dir {
			entryPath := path.Join(dirOrFile, info.Name())
			err := g.ReadDirOrFile(entryPath, data)
			if err != nil {
				return fmt.Errorf("g.ReadDirOrFile(), err:%w", err)
			}
		}
		return nil
	}
	openFile, err := g.ReadFile(dirOrFile)
	if err != nil {
		return fmt.Errorf("g.ReadFile(), err:%w", err)
	}
	data[dirOrFile] = openFile
	return nil
}

// CommitID
//
//	@Description: 获取 CommitID
//	@receiver g
//	@return string
//	@return error
func (g *Git) CommitID() (string, error) {
	revision, err := g.Repository.ResolveRevision("HEAD")
	if err != nil {
		return "", fmt.Errorf("g.Repository.ResolveRevision(), err:%w", err)
	}
	return revision.String(), nil
}

// CommitID2Short
//
//	@Description: 获取 CommitID 的前 8 位
//	@receiver g
//	@return string
//	@return error
func (g *Git) CommitID2Short() (string, error) {
	id, err := g.CommitID()
	if err != nil {
		return "", err
	}
	return id[:8], nil
}
