package pkg

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Command interface {
	Execute() (string, error)
}

type Copy interface {
	DownloadFiles() (string, error)
	UploadFiles() (string, error)
}

type SSHConfig struct {
	Host           string
	User           string
	PrivateKeyPath string
	Password       string // 新增密码字段
	TimeOut        time.Duration
}

type ServiceInfo struct {
	Host    string
	Command string
}

type Message struct {
	Result string
	Error  error
}

type ExecuteResult struct {
	Host    string
	Command string
	Message Message
}

type CopyResult struct {
	RemoteHost string
	RemotePath string
	LocalPath  string
	Message    Message
}

type SSHCommand struct {
	Config      SSHConfig
	Client      *ssh.Client
	ServiceInfo ServiceInfo
}

func (s *SSHCommand) SSHClient() (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if s.Config.Password != "" {
		authMethods = append(authMethods, ssh.Password(s.Config.Password))
	} else {
		key, err := ioutil.ReadFile(s.Config.PrivateKeyPath)
		if err != nil {
			return nil, err
		}

		sign, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}

		authMethods = append(authMethods, ssh.PublicKeys(sign))
	}

	config := &ssh.ClientConfig{
		User:            s.Config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         s.Config.TimeOut,
	}

	client, err := ssh.Dial("tcp", s.Config.Host+":22", config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %s", err)
	}
	return client, nil
}

func (s *SSHCommand) Execute() (string, error) {
	if s.Client == nil {
		client, err := s.SSHClient()
		if err != nil {
			return "", err
		}
		s.Client = client
	}

	session, err := s.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %s", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(s.ServiceInfo.Command)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

type SCPCommand struct {
	Config     SSHConfig
	LocalPath  string
	RemotePath string
}

func (s *SCPCommand) SSHClient() (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if s.Config.Password != "" {
		authMethods = append(authMethods, ssh.Password(s.Config.Password))
	} else {
		key, err := ioutil.ReadFile(s.Config.PrivateKeyPath)
		if err != nil {
			return nil, err
		}

		sign, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}

		authMethods = append(authMethods, ssh.PublicKeys(sign))
	}

	config := &ssh.ClientConfig{
		User:            s.Config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         s.Config.TimeOut,
	}

	client, err := ssh.Dial("tcp", s.Config.Host+":22", config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %s", err)
	}
	return client, nil
}

func (s *SCPCommand) DownloadFiles() (string, error) {
	client, err := s.SSHClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", err
	}
	defer sftpClient.Close()

	err = s.download(sftpClient)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SFTP download from %s successful", s.Config.Host), nil
}

func (s *SCPCommand) UploadFiles() (string, error) {
	client, err := s.SSHClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", err
	}
	defer sftpClient.Close()

	err = s.upload(sftpClient)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SFTP upload to %s successful", s.Config.Host), nil

}

func (s *SCPCommand) upload(sftpClient *sftp.Client) error {
	return filepath.Walk(s.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(s.LocalPath, path)
		if err != nil {
			return err
		}

		remotePath := filepath.Join(s.RemotePath, relPath)

		if info.IsDir() {
			if err := sftpClient.MkdirAll(filepath.Dir(remotePath)); err != nil {
				return err
			}
		} else {

			if err := sftpClient.MkdirAll(filepath.Dir(remotePath)); err != nil {
				return err
			}

			localFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer localFile.Close()

			remoteFile, err := sftpClient.Create(remotePath)
			if err != nil {
				return err
			}
			defer remoteFile.Close()

			if _, err := io.Copy(remoteFile, localFile); err != nil {
				return err
			}
			if err := sftpClient.Chmod(remotePath, info.Mode()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SCPCommand) download(sftpClient *sftp.Client) error {
	walker := sftpClient.Walk(s.RemotePath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		relPath, err := filepath.Rel(s.RemotePath, walker.Path())
		if err != nil {
			return err
		}

		localPath := filepath.Join(s.LocalPath, relPath)

		if walker.Stat().IsDir() {
			if err := os.MkdirAll(filepath.Dir(localPath), os.ModePerm); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(localPath), os.ModePerm); err != nil {
				return err
			}

			localFile, err := os.Create(localPath)
			if err != nil {
				return err
			}
			defer localFile.Close()

			remoteFile, err := sftpClient.Open(walker.Path())
			if err != nil {
				return err
			}
			defer remoteFile.Close()

			if _, err := io.Copy(localFile, remoteFile); err != nil {
				return err
			}

			if err := os.Chmod(localPath, walker.Stat().Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

type Invoker struct {
	Commands   []Command
	Copy       []Copy
	Result     ExecuteResult
	CopyResult CopyResult
	mu         sync.Mutex
	wg         sync.WaitGroup
}

func (i *Invoker) AddCommand(cmd Command) {
	i.Commands = append(i.Commands, cmd)
}

func (i *Invoker) AddCopyCommand(copy Copy) {
	i.Copy = append(i.Copy, copy)
}

func (i *Invoker) ExecuteCommand() {
	for _, cmd := range i.Commands {
		i.wg.Add(1)
		go func(cmd Command) {
			defer i.wg.Done()
			result, err := cmd.Execute()
			if err != nil {
				i.Result = ExecuteResult{
					Host:    cmd.(*SSHCommand).ServiceInfo.Host,
					Command: cmd.(*SSHCommand).ServiceInfo.Command,
					Message: Message{
						Result: "",
						Error:  err,
					},
				}
			}

			i.mu.Lock()
			defer i.mu.Unlock()
			i.Result = ExecuteResult{
				Host:    cmd.(*SSHCommand).ServiceInfo.Host,
				Command: cmd.(*SSHCommand).ServiceInfo.Command,
				Message: Message{
					Result: result,
					Error:  nil,
				},
			}
		}(cmd)
	}
	i.wg.Wait()
}

func (i *Invoker) ScpDownloadFiles() {
	for _, cmd := range i.Copy {
		i.wg.Add(1)
		go func(cmd Copy) {
			defer i.wg.Done()
			result, err := cmd.DownloadFiles()
			if err != nil {
				i.CopyResult = CopyResult{
					RemoteHost: cmd.(*SCPCommand).Config.Host,
					RemotePath: cmd.(*SCPCommand).RemotePath,
					LocalPath:  cmd.(*SCPCommand).LocalPath,
					Message: Message{
						Result: "",
						Error:  err,
					},
				}
			}

			i.mu.Lock()
			defer i.mu.Unlock()
			i.CopyResult = CopyResult{
				RemoteHost: cmd.(*SCPCommand).Config.Host,
				RemotePath: cmd.(*SCPCommand).RemotePath,
				LocalPath:  cmd.(*SCPCommand).LocalPath,
				Message: Message{
					Result: result,
					Error:  nil,
				},
			}
		}(cmd)
	}
	i.wg.Wait()
}

func (i *Invoker) ScpUploadFiles() {
	for _, cmd := range i.Copy {
		i.wg.Add(1)
		go func(cmd Copy) {
			defer i.wg.Done()
			result, err := cmd.UploadFiles()
			if err != nil {
				i.CopyResult = CopyResult{
					RemoteHost: cmd.(*SCPCommand).Config.Host,
					RemotePath: cmd.(*SCPCommand).RemotePath,
					LocalPath:  cmd.(*SCPCommand).LocalPath,
					Message: Message{
						Result: "",
						Error:  err,
					},
				}
			}

			i.mu.Lock()
			defer i.mu.Unlock()
			i.CopyResult = CopyResult{
				RemoteHost: cmd.(*SCPCommand).Config.Host,
				RemotePath: cmd.(*SCPCommand).RemotePath,
				LocalPath:  cmd.(*SCPCommand).LocalPath,
				Message: Message{
					Result: result,
					Error:  nil,
				},
			}
		}(cmd)
	}
	i.wg.Wait()
}

type CommandFactory struct {
	SSHConfig SSHConfig
}

// InitTask
//
//	@Description:
//	@param SSHConfig
//	@return CommandFactory 命令工厂
//	@return *Invoker 调用者
func InitTask(SSHConfig SSHConfig) (CommandFactory, *Invoker) {
	remoteHost := CommandFactory{SSHConfig: SSHConfig}
	return remoteHost, &Invoker{}
}

func (cf *CommandFactory) CreateExecuteCommand(command string) Command {
	return &SSHCommand{
		ServiceInfo: ServiceInfo{
			Command: command,
			Host:    cf.SSHConfig.Host,
		},
		Config: cf.SSHConfig,
	}
}

func (cf *CommandFactory) CreateDownloadCommand(localPath, remotePath string) Copy {
	return &SCPCommand{
		Config:     cf.SSHConfig,
		LocalPath:  localPath,
		RemotePath: remotePath,
	}
}

func (cf *CommandFactory) CreateUploadCommand(localPath, remotePath string) Copy {
	return &SCPCommand{
		Config:     cf.SSHConfig,
		LocalPath:  localPath,
		RemotePath: remotePath,
	}
}
