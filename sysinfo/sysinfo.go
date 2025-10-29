package sysinfo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/cpuid/v2"
	"github.com/minio/dperf/pkg/dperf"
	"github.com/opencontainers/selinux/go-selinux"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/youcd/toolkit/net"
	"github.com/youcd/toolkit/sysinfo/types"
	"github.com/youcd/toolkit/systemd"
	"go.uber.org/zap/buffer"
	"golang.org/x/sys/unix"
)

var (
	pwd string
	wg  sync.WaitGroup
)
var (
	ErrInstallDirIsNotDir = errors.New("安装目录不是目录")
	ErrMkDirInstallDir    = errors.New("创建安装目录失败")
	ErrPerfRun            = errors.New("性能测试失败")
	ErrPerfResultIsNull   = errors.New("性能测试结果为空")
	ErrCheckCGroupVersion = errors.New("检查cgroup版本失败")
	ErrGetenforceNotExist = errors.New("getenforce命令不存在")
)

func SysInfo(ctx context.Context, skipDiskPerformance bool, installDirs []string, services []string) (*types.Host, error) {
	var errs []error
	h := new(types.Host)

	IP := &types.IP{
		Address: "127.0.0.1",
		Port:    0,
		IPType:  0,
	}
	h.IP = IP
	h.DataDiskFree = make(map[string]uint64)
	// 磁盘使用情况
	for _, dir := range installDirs {
		stat, err := os.Stat(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err = os.MkdirAll(dir, 755)
				if err != nil {
					errs = append(errs, ErrMkDirInstallDir)
				}
			} else {
				errs = append(errs, err)
			}
		}

		if stat != nil && !stat.IsDir() {
			errs = append(errs, ErrInstallDirIsNotDir)
		}
		// 检查磁盘性能  diskPerformance
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			if !skipDiskPerformance {
				errs = append(errs, diskPerformance(ctx, h, dir))
			}
		}(dir)
		// 检查磁盘空间
		usage, err := disk.Usage(dir)
		if err == nil {
			h.DataDiskFree[dir] = usage.Free
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	// 总内存
	memory, err := mem.VirtualMemory()
	if err == nil {
		h.MemorySize = memory.Total
	}
	// 系统架构
	arch, err := host.KernelArch()
	if err == nil {
		h.Arch = arch
	}
	// 系统信息
	info, err := host.Info()
	if err == nil {
		h.HostInfo = info
	}

	// getSelinux
	getSelinux(h)

	//  获取主机时间
	timeFetch(h)

	err = sudo(ctx, h)
	if err != nil {
		errs = append(errs, err)
	}

	//  CPU 配置
	Count, err := cpu.Counts(true)
	if err != nil {
		errs = append(errs, err)
	}
	h.CPUCount = Count

	// bpffs 检查
	checkBPFfs(h)

	// service 检查
	err = service(ctx, h, services)
	if err != nil {
		errs = append(errs, err)
	}

	// CGroup版本
	err = cGroupVersion(h)
	if err != nil {
		errs = append(errs, err)
	}

	// multiIP
	multiIP(h)

	h.CPUInfo = &cpuid.CPU

	wg.Wait()

	if len(errs) > 0 {
		return h, errors.Join(errs...)
	}

	return h, nil
}

// checkBPFfs
//
//	@Description: bpf file system
//	@param t
func checkBPFfs(t *types.Host) {
	_, err := os.Stat("/sys/fs/bpf")
	if err != nil {
		t.BPFFSCheck = false
		return
	}
	t.BPFFSCheck = true
}

var (
	allCommand      = regexp.MustCompile(` ?\(ALL\) ? ALL`)
	allCommandNoPWD = regexp.MustCompile(` ?\(ALL\) NOPASSWD: ALL`)
)

// sudo
//
//	@Description:sudo检查
//	@param t
//	@return error
func sudo(ctx context.Context, t *types.Host) error {
	whoami, err := exec.CommandContext(ctx, "whoami").Output()
	if err != nil {
		return fmt.Errorf("whoami error: %w", err)
	}

	s := strings.Trim(string(whoami), "\n")

	if strings.ToLower(s) == "root" {
		t.Sudo = true
		return nil
	}

	cmd := exec.CommandContext(ctx, "sudo", "-S", "-l")
	cmd.Stdin = strings.NewReader(pwd + "\n")
	var result buffer.Buffer
	cmd.Stdout = &result
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("sudo -S -l: %w", err)
	}
	trimSpace := strings.TrimSpace(result.String())
	l := strings.Split(trimSpace, "\n")
	if len(l) == 0 {
		t.Sudo = true
		return nil
	}
	sudoStr := l[len(l)-1]
	if allCommand.MatchString(sudoStr) || allCommandNoPWD.MatchString(sudoStr) {
		t.Sudo = true
		return nil
	}
	return nil
}

// getSelinux
//
//	@Description: selinux信息提取
//	@param h
//	@return error
func getSelinux(h *types.Host) {
	enabled := selinux.GetEnabled()
	if enabled {
		h.Selinux = types.SelinuxEnforcing
		return
	}
	mode := selinux.EnforceMode()
	if mode == 0 {
		h.Selinux = types.SelinuxPermissive
		return
	}
	h.Selinux = types.SelinuxDisabled
	return
}

// timeFetch
//
//	@Description: 时间提取
//	@param h
func timeFetch(h *types.Host) {
	h.CurentTime = time.Now().Unix()
}

// Service
//
//	@Description:检查服务
//	@param h
//	@return string
//	@return bool
func service(ctx context.Context, h *types.Host, services []string) error {
	var errs []error
	newSystemd, err := systemd.NewSystemd(ctx, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("systemd new error: %w", err)
	}
	for _, s := range services {
		l, err := newSystemd.UnitListFilterByName(ctx, s)
		if err != nil {
			if errors.Is(err, systemd.ErrServiceNotExist) {
				continue
			}
			errs = append(errs, err)
		}
		h.Services = append(h.Services, l)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func cGroupVersion(h *types.Host) error {
	mode := Mode()
	switch mode {
	case Unavailable:
		h.CGroupVersion = types.CGroupVersionUnavailable
		return nil
	case Legacy:
		h.CGroupVersion = types.CGroupVersionLegacy
		return nil
	case Hybrid:
		h.CGroupVersion = types.CGroupVersionHybrid
		return nil
	case Unified:
		h.CGroupVersion = types.CGroupVersionUnified
		return nil
	default:
		return fmt.Errorf("cgroup version error: %w", ErrCheckCGroupVersion)
	}
}

type CGMode int

const (
	// Unavailable cgroup mountpoint
	Unavailable CGMode = iota
	// Legacy cgroups v1
	Legacy
	// Hybrid with cgroups v1 and v2 controllers mounted
	Hybrid
	// Unified with only cgroups v2 mounted
	Unified
)

const unifiedMountpoint = "/sys/fs/cgroup"

func Mode() CGMode {
	var cgMode CGMode

	var st unix.Statfs_t
	err := unix.Statfs(unifiedMountpoint, &st)
	if err != nil {
		cgMode = Unavailable
		return cgMode
	}
	switch st.Type {
	case unix.CGROUP2_SUPER_MAGIC:
		cgMode = Unified
	default:
		cgMode = Legacy
		err := unix.Statfs(filepath.Join(unifiedMountpoint, "unified"), &st)
		if err != nil {
			return cgMode
		}
		if st.Type == unix.CGROUP2_SUPER_MAGIC {
			cgMode = Hybrid
		}
	}
	return cgMode
}

// diskPerformance
//
//	@Description: 检查此磁盘性能
//	@param h
func diskPerformance(ctx context.Context, h *types.Host, performanceDir string) error {
	// 默认值,防止空指针 panic
	h.DrivePerfResult = make(map[string]*dperf.DrivePerfResult)
	perf := &dperf.DrivePerf{
		Serial:     false,
		BlockSize:  4194304,    // 4MiB
		FileSize:   1073741824, // 1GiB
		Verbose:    false,
		IOPerDrive: 4,
	}
	results, err := perf.Run(ctx, performanceDir)
	if err != nil {
		return ErrPerfRun
	}
	if len(results) == 0 {
		return ErrPerfResultIsNull
	}
	// 重新赋值
	h.DrivePerfResult[performanceDir] = results[0]
	return nil
}

// multiIP
//
//	@Description: 多网卡
//	@param h
func multiIP(h *types.Host) {
	address, err := net.PhysicsCNIAddress()
	if err != nil {
		h.IPs = []string{}
		return
	}
	h.IPs = address
}
