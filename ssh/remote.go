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
	Copy() (string, error)
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

type ExecuteResult struct {
	Host    string
	Command string
	Result  string
}

type ExecuteCheckResult struct {
	ServiceName string
	Result      string
}

type CopyResult struct {
	Host       string
	RemoteHost string
	RemotePath string
	LocalPath  string
	Result     string
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

func (s *SSHCommand) Copy() (string, error) {
	return "", fmt.Errorf("SSHCommand does not implement Copy")
}

type SCPCommand struct {
	Config     SSHConfig
	LocalPath  string
	RemotePath string
	Upload     bool // 上传为 true, 下载为 false
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

func (s *SCPCommand) Execute() (string, error) {
	return "", fmt.Errorf("SCPCommand does not implement Execute")
}

func (s *SCPCommand) Copy() (string, error) {
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

	if s.Upload {
		err = s.upload(sftpClient)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("SFTP upload to %s successful", s.Config.Host), nil
	} else {
		err = s.download(sftpClient)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("SFTP download from %s successful", s.Config.Host), nil
	}
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
			if err := sftpClient.MkdirAll(remotePath); err != nil {
				return err
			}
		} else {
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
			if err := os.MkdirAll(localPath, os.ModePerm); err != nil {
				return err
			}
		} else {
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
		}
	}
	return nil
}

type Invoker struct {
	Commands    []Command
	CheckResult map[string][]ExecuteCheckResult
	Result      ExecuteResult
	CopyResult  CopyResult
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func (i *Invoker) AddCommand(cmd Command) {
	i.Commands = append(i.Commands, cmd)
}

func (i *Invoker) ExecuteCommand() {
	for _, cmd := range i.Commands {
		i.wg.Add(1)
		go func(cmd Command) {
			defer i.wg.Done()
			result, err := cmd.Execute()
			if err != nil {
				command, host := ExecuteArgs(cmd)
				i.Result = ExecuteResult{
					Host:    host,
					Command: command,
					Result:  fmt.Sprintf("Failed: %v", err.Error()),
				}
			}

			i.mu.Lock()
			defer i.mu.Unlock()
			command, host := ExecuteArgs(cmd)
			i.Result = ExecuteResult{
				Host:    host,
				Command: command,
				Result:  result,
			}
		}(cmd)
	}
	i.wg.Wait()
}

func (i *Invoker) CopyFiles() {
	for _, cmd := range i.Commands {
		i.wg.Add(1)
		go func(cmd Command) {
			defer i.wg.Done()
			result, err := cmd.Copy()
			if err != nil {
				args := CopyArgs(cmd)
				i.CopyResult = CopyResult{
					Host:       args.Host,
					RemoteHost: args.RemoteHost,
					RemotePath: args.RemotePath,
					LocalPath:  args.LocalPath,
					Result:     fmt.Sprintf("Scp failed: %s", err.Error()),
				}
			}

			i.mu.Lock()
			defer i.mu.Unlock()
			args := CopyArgs(cmd)
			i.CopyResult = CopyResult{
				Host:       args.Host,
				RemoteHost: args.RemoteHost,
				RemotePath: args.RemotePath,
				LocalPath:  args.LocalPath,
				Result:     result,
			}
		}(cmd)
	}
	i.wg.Wait()
}

func ExecuteArgs(cmd Command) (command, host string) {
	if executeCmd, ok := cmd.(*SSHCommand); ok {
		return executeCmd.ServiceInfo.Command, executeCmd.ServiceInfo.Host
	}
	return "unknown", "unknown"
}

func CopyArgs(cmd Command) CopyResult {
	if copyCmd, ok := cmd.(*SCPCommand); ok {
		return CopyResult{
			Host:       copyCmd.Config.Host,
			RemoteHost: copyCmd.RemotePath,
			RemotePath: copyCmd.RemotePath,
			LocalPath:  copyCmd.LocalPath,
		}
	}
	return CopyResult{}
}

type CommandFactory struct {
	SSHConfig SSHConfig
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

func (cf *CommandFactory) CreateSCPCommand(localPath, remotePath string, upload bool) Command {
	return &SCPCommand{
		Config:     cf.SSHConfig,
		LocalPath:  localPath,
		RemotePath: remotePath,
		Upload:     upload,
	}
}

// 使用demo
//func main() {
//	sshConfig1 := SSHConfig{
//		Host: "192.168.111.232",
//		User: "root",
//		//PrivateKeyPath: "/root/.ssh/id_rsa",
//		Password: "123457.ii",
//	}
//
//	HostOne := CommandFactory{SSHConfig: sshConfig1}
//	//HostTwo := CommandFactory{SSHConfig: sshConfig2}
//
//	invoker := &Invoker{}
//
//	filesToUpload := map[string]string{
//		"/root/proxy.sh": "/root/proxy.sh",
//	}
//
//	filesToDownload := map[string]string{
//		"/root/config": "/root/config",
//	}
//
//	for local, remote := range filesToUpload {
//		invoker.AddCommand(HostOne.CreateSCPCommand(local, remote, true))
//	}
//
//	for remote, local := range filesToDownload {
//		invoker.AddCommand(HostOne.CreateSCPCommand(local, remote, false))
//	}
//
//	invoker.AddCommand(HostOne.CreateCheckCommand("upgrade", "systemctl is-active upgrade"))
//	invoker.AddCommand(HostOne.CreateCheckCommand("license", "systemctl is-active license"))
//	invoker.AddCommand(HostOne.CreateCheckCommand("platform", "systemctl is-active platform"))
//
//	//invoker.AddCommand(HostTwo.CreateCheckCommand("upgrade", "systemctl is-active upgrade"))
//	//invoker.AddCommand(HostTwo.CreateCheckCommand("license", "systemctl is-active license"))
//	//invoker.AddCommand(HostTwo.CreateCheckCommand("platform", "systemctl is-active platform"))
//	invoker.AddCommand(HostOne.CreateExecuteCommand("ls /tmp && "))
//
//	invoker.ExecuteCheckCommand()
//	invoker.ExecuteCommand()
//	invoker.CopyFiles()//
//	fmt.Println()
//	for k, v := range invoker.CheckResult {
//		fmt.Printf("Host: %s, Results:\n", k)
//		for _, result := range v {
//			fmt.Printf("  Service: %s, Result: %s\n", result.ServiceName, result.Result)
//		}
//	}
//
//	fmt.Printf("Host: %s, Command: %s, Result: %s\n", invoker.Result.Host, invoker.Result.Command, invoker.Result.Result)
//	fmt.Println()
//	fmt.Printf("localhost: %s\n,remotehost: %s\n,localPath: %s\n,remotePath: %s\n, result: %s\n",
//		invoker.CopyResult.Host,
//		invoker.CopyResult.RemoteHost,
//		invoker.CopyResult.LocalPath,
//		invoker.CopyResult.RemotePath,
//		invoker.CopyResult.Result,
//	)
//
//}
