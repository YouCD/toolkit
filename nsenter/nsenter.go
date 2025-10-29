package nsenter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// Config is the nsenter configuration used to generate
// nsenter command
type Config struct {
	Cgroup              bool   // Enter cgroup namespace
	CgroupFile          string // Cgroup namespace location, default to /proc/PID/ns/cgroup
	FollowContext       bool   // Set SELinux security context
	GID                 int    // GID to use to execute given program
	IPC                 bool   // Enter IPC namespace
	IPCFile             string // IPC namespace location, default to /proc/PID/ns/ipc
	Mount               bool   // Enter mount namespace
	MountFile           string // Mount namespace location, default to /proc/PID/ns/mnt
	Net                 bool   // Enter network namespace
	NetFile             string // Network namespace location, default to /proc/PID/ns/net
	NoFork              bool   // Do not fork before executing the specified program
	PID                 bool   // Enter PID namespace
	PIDFile             string // PID namespace location, default to /proc/PID/ns/pid
	PreserveCredentials bool   // Preserve current UID/GID when entering namespaces
	RootDirectory       string // Set the root directory, default to target process root directory
	Target              int    // Target PID (required)
	UID                 int    // UID to use to execute given program
	User                bool   // Enter user namespace
	UserFile            string // User namespace location, default to /proc/PID/ns/user
	UTS                 bool   // Enter UTS namespace
	UTSFile             string // UTS namespace location, default to /proc/PID/ns/uts
	WorkingDirectory    string // Set the working directory, default to target process working directory
}

// ExecuteContext
//
//	@Description: 执行命令
//	@receiver c
//	@param ctx
//	@param commd
//	@param commdArgs
//	@return string
//	@return string
//	@return error
func (c *Config) ExecuteContext(ctx context.Context, commd string, commdArgs ...string) (string, string, error) {
	cmd := c.buildCommand(ctx)

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Args = append(cmd.Args, commd)
	cmd.Args = append(cmd.Args, commdArgs...)

	err := cmd.Run()
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("cmd.Run(): %w", err)
	}

	return stdout.String(), stderr.String(), nil
}

// Enter
//
//	@Description: 执行 nsenter
//	@receiver c
//	@param ctx
//	@return error
func (c *Config) Enter(ctx context.Context) error {
	cmd := c.buildCommand(ctx)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	//nolint:wrapcheck
	return cmd.Run()
}

// buildCommand
//
//	@Description: 构建 nsenter
//	@receiver c
//	@param ctx
//	@return *exec.Cmd
//	@return error
func (c *Config) buildCommand(ctx context.Context) *exec.Cmd {
	var args = []string{"nsenter"}

	args = append(args, "--target", strconv.Itoa(c.Target))

	if c.Cgroup {
		args = append(args, "--cgroup")
		if c.CgroupFile != "" {
			args = append(args, c.CgroupFile)
		}
	}

	if c.FollowContext {
		args = append(args, "--follow-context")
	}

	if c.GID != 0 {
		args = append(args, "--setgid", strconv.Itoa(c.GID))
	}

	if c.IPC {
		args = append(args, "--ipc")
		if c.IPCFile != "" {
			args = append(args, c.IPCFile)
		}
	}

	if c.Mount {
		args = append(args, "--mount")
		if c.MountFile != "" {
			args = append(args, c.MountFile)
		}
	}

	if c.Net {
		args = append(args, "--net")
		if c.NetFile != "" {
			args = append(args, c.NetFile)
		}
	}

	if c.NoFork {
		args = append(args, "--no-fork")
	}

	if c.PID {
		args = append(args, "--pid")
		if c.PIDFile != "" {
			args = append(args, c.PIDFile)
		}
	}

	if c.PreserveCredentials {
		args = append(args, "--preserve-credentials")
	}

	if c.RootDirectory != "" {
		args = append(args, "--root", c.RootDirectory)
	}

	if c.UID != 0 {
		args = append(args, "--setuid", strconv.Itoa(c.UID))
	}

	if c.User {
		args = append(args, "--user")
		if c.UserFile != "" {
			args = append(args, c.UserFile)
		}
	}

	if c.UTS {
		args = append(args, "--uts")
		if c.UTSFile != "" {
			args = append(args, c.UTSFile)
		}
	}

	if c.WorkingDirectory != "" {
		args = append(args, "--wd", c.WorkingDirectory)
	}

	return exec.CommandContext(ctx, "nsenter", args...)
}
