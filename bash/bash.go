package bash

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

var (
	ErrBashExec = errors.New("bash执行错误")
)

// Bash
//
//	@Description: 执行bash命令
//	@param cmd
//	@return out
//	@return exitCode
func Bash(cmd string) (string, int) {
	if cmd == "" {
		//log.Debug("cmd命令为空")
		return "", 0
	}
	//log.Debugw("cmd", "cmd", cmd)
	command := exec.Command("bash", "-c", cmd)
	command.Env = append(command.Environ(), "LANG=en_US.utf8", "LANGUAGE=en_US.utf8")
	output, err := command.CombinedOutput()
	if err != nil {
		//nolint:errorlint
		if ins, ok := err.(*exec.ExitError); ok {
			out := string(output)
			exitCode := ins.ExitCode()
			return out, exitCode
		}
		return err.Error(), 1
	}
	return string(output), 0
}
func BashFormat(format string, a ...any) (string, int) {
	cmd := fmt.Sprintf(format, a...)
	if cmd == "" {
		//log.Debug("cmd命令为空")
		return "", 0
	}
	//log.Debugw("cmd", "cmd", cmd)
	command := exec.Command("bash", "-c", cmd)
	command.Env = append(command.Environ(), "LANG=en_US.utf8", "LANGUAGE=en_US.utf8")
	output, err := command.CombinedOutput()
	if err != nil {
		//nolint:errorlint
		if ins, ok := err.(*exec.ExitError); ok {
			out := string(output)
			exitCode := ins.ExitCode()
			return out, exitCode
		}
		return err.Error(), 1
	}
	return string(output), 0
}

// BashWithWorkDir
//
//	@Description: 指定工作目录执行bash命令
//	@param cmd
//	@param dir
//	@return out
//	@return exitCode
func BashWithWorkDir(cmd, dir string) (string, int) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Sprintf("工作目录创建失败:%s", dir), 1
	}
	//log.Debugw("执行bash命令", "cmd", cmd, "dir", dir)
	command := exec.Command("bash", "-c", cmd)
	command.Dir = dir
	command.Env = append(command.Environ(), "LANG=en_US.utf8", "LANGUAGE=en_US.utf8")
	output, err := command.CombinedOutput()
	if err != nil {
		//nolint:errorlint
		if ins, ok := err.(*exec.ExitError); ok {
			out := string(output)
			exitCode := ins.ExitCode()
			return out, exitCode
		}
		return err.Error(), 1
	}
	return string(output), 0
}

func IsRootUser() (string, bool) {
	u, _ := user.Current()
	if u.Username == "root" {
		return "root", true
	}
	return u.Name, false
}

// CMDExecMySQLScript
//
//	@Description: SQL脚本升级
//	@param sqlFile
//	@return error
func CMDExecMySQLScript(sqlFile string) error {
	if out, code := Bash("/usr/local/docker/docker exec -i mysql sh -c 'exec mysql -uroot -pMysql@ms2016 '< " + sqlFile); code != 0 {
		//log.Error(out)
		return fmt.Errorf("执行SQL失败，out:%s, err:%w", out, ErrBashExec)
	}
	return nil
}
