package bash

import (
	"context"
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
func Bash(ctx context.Context, cmd string) (string, int) {
	if cmd == "" {
		return "", 0
	}
	command := exec.CommandContext(ctx, "bash", "-c", cmd)
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
func BashFormat(ctx context.Context, format string, a ...any) (string, int) {
	cmd := fmt.Sprintf(format, a...)
	if cmd == "" {
		return "", 0
	}
	command := exec.CommandContext(ctx, "bash", "-c", cmd)
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
func BashWithWorkDir(ctx context.Context, cmd, dir string) (string, int) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Sprintf("工作目录创建失败:%s", dir), 1
	}
	command := exec.CommandContext(ctx, "bash", "-c", cmd)
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
