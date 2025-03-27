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

	"github.com/hairyhenderson/go-which"
	"github.com/minio/dperf/pkg/dperf"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"gitlab.firecloud.wan/devops/ops-toolkit/net"
	"gitlab.firecloud.wan/devops/ops-toolkit/sysinfo/types"
	"gitlab.firecloud.wan/devops/ops-toolkit/systemd"
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

func SysInfo(ctx context.Context, skipDiskPerformance bool, installDir string, services []string) (*types.Host, error) {
	var errs []error
	h := new(types.Host)

	IP := &types.IP{
		Address: "127.0.0.1",
		Port:    0,
		IPType:  0,
	}
	h.IP = IP
	// 磁盘使用情况
	stat, err := os.Stat(installDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.MkdirAll(installDir, 755); err != nil {
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
	go func() {
		defer wg.Done()
		if !skipDiskPerformance {
			errs = append(errs, diskPerformance(ctx, h, installDir))
		}
	}()

	usage, err := disk.Usage(installDir)
	if err == nil {
		h.DataDiskFree = usage.Free
		errs = append(errs, err)
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

	// selinux
	errs = append(errs, selinux(h))

	//  获取主机时间
	timeFetch(h)

	errs = append(errs, sudo(h))

	//  CPU 配置
	Count, err := cpu.Counts(true)
	if err != nil {
		errs = append(errs, err)
	}
	h.CPUCount = Count

	// bpffs 检查
	checkBPFfs(h)

	// service 检查
	errs = append(errs, service(ctx, h, services))

	// CGroup版本
	errs = append(errs, cGroupVersion(h))

	// multiIP
	multiIP(h)

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
	if _, err := os.Stat("/sys/fs/bpf"); err != nil {
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
func sudo(t *types.Host) error {
	whoami, err := exec.Command("whoami").Output()
	if err != nil {
		return fmt.Errorf("whoami error: %w", err)
	}

	s := strings.Trim(string(whoami), "\n")

	if strings.ToLower(s) == "root" {
		t.Sudo = true
		return nil
	}

	cmd := exec.Command("sudo", "-S", "-l")
	cmd.Stdin = strings.NewReader(pwd + "\n")
	var result buffer.Buffer
	cmd.Stdout = &result
	if err = cmd.Run(); err != nil {
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

// selinux
//
//	@Description: selinux信息提取
//	@param h
//	@return error
func selinux(h *types.Host) error {
	switch h.Platform() {
	case types.OSPlatformCentos, types.OSPlatformBigCloud, types.OSPlatformAnolis, types.OSPlatformOpeneuler, types.OSPlatformUOS, types.OSPlatformRocky, types.OSPlatformKylin:
		getenforcePath := which.Which("getenforce")
		if getenforcePath != "" {
			selinuxResult, err := exec.Command("getenforce").Output()
			if err != nil {
				return fmt.Errorf("getenforce error: %w", err)
			}
			s := strings.Trim(string(selinuxResult), "\n")
			h.Selinux = types.NewSelinux(s)
			return nil
		}
		return ErrGetenforceNotExist
	case types.OSPlatformUbuntu, types.OSPlatformOpensuseLeap:
		h.Selinux = types.SelinuxDisabled
	}
	return nil
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
	if err := unix.Statfs(unifiedMountpoint, &st); err != nil {
		cgMode = Unavailable
		return cgMode
	}
	switch st.Type {
	case unix.CGROUP2_SUPER_MAGIC:
		cgMode = Unified
	default:
		cgMode = Legacy
		if err := unix.Statfs(filepath.Join(unifiedMountpoint, "unified"), &st); err != nil {
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
	drivePerfResult := new(dperf.DrivePerfResult)
	// 默认值,防止空指针 panic
	h.DrivePerfResult = drivePerfResult
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
	h.DrivePerfResult = results[0]
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
