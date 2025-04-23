package types

import (
	"strings"

	"github.com/klauspost/cpuid/v2"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/minio/dperf/pkg/dperf"
	"github.com/shirou/gopsutil/v3/host"
)

type (
	CGroupVersion string
	OSArch        string
	OSType        string
)

const (
	CGroupVersionLegacy      CGroupVersion = "Legacy"
	CGroupVersionHybrid      CGroupVersion = "Hybrid"
	CGroupVersionUnified     CGroupVersion = "Unified"
	CGroupVersionUnavailable CGroupVersion = "Unavailable"

	OSArchX8664 OSArch = "x86_64"
	OSArchOther OSArch = "other"

	OSTypeLinux OSType = "linux"
	OSTypeOther OSType = "other"
)

func (c CGroupVersion) String() string {
	return string(c)
}

type CheckItem struct {
	Name    string // 名称
	Require string // 要求
	Reason  string // 原因
	IsForce bool   // 强制要求
	IsPass  bool   // 结果
}

type Host struct {
	IP              *IP                    `json:"ip"`
	User            string                 `json:"user"`
	Password        string                 `json:"password"`
	Arch            string                 `json:"arch"`
	MemorySize      uint64                 `json:"memorySize"` // Bytes
	HostInfo        *host.InfoStat         `json:"hostInfo"`
	DataDiskFree    uint64                 `json:"free"` // Bytes
	Services        []*dbus.UnitStatus     `json:"services"`
	Hostname        string                 `json:"hostname"`
	Selinux         Selinux                `json:"selinux"`
	OutputInterface string                 `json:"outputInterface"`
	ResolvConfg     bool                   `json:"resolvConfg"`
	CurentTime      int64                  `json:"currentTime"`
	CPUCount        int                    `json:"cpuCount"`
	Sudo            bool                   `json:"sudo"`
	BPFFSCheck      bool                   `json:"bpffsCheck"`
	CGroupVersion   CGroupVersion          `json:"cgroupVersion"`
	DrivePerfResult *dperf.DrivePerfResult `json:"drivePerfResult"`
	IPs             []string               `json:"IPs"` //nolint:tagliatelle
	CPUInfo         *cpuid.CPUInfo         `json:"CPUInfo"`
}

func (h *Host) Platform() OSPlatform {
	switch strings.ToLower(h.HostInfo.Platform) {
	case "centos":
		return OSPlatformCentos
	case "ubuntu":
		return OSPlatformUbuntu
	case "bigcloud":
		return OSPlatformBigCloud
	case "anolis":
		return OSPlatformAnolis
	case "openeuler":
		return OSPlatformOpeneuler
	case "uos":
		return OSPlatformUOS
	case "opensuse-leap":
		return OSPlatformOpensuseLeap
	case "rocky":
		return OSPlatformRocky
	case "kylin":
		return OSPlatformKylin
	default:
		return OSPlatformOther
	}
}

// Str2OsArch
//
//	@Description:字符串转化为 Arch
//	@param arch
//	@return OSArch
func (h *Host) Str2OsArch() OSArch {
	switch strings.ToLower(h.Arch) {
	case "x86_64":
		return OSArchX8664
	default:
		return OSArchOther
	}
}

// Str2OsType
//
//	@Description: 字符串转化为 OSType
//	@param os
//	@return OSType
func (h *Host) Str2OsType() OSType {
	switch strings.ToLower(h.HostInfo.OS) {
	case "linux":
		return OSTypeLinux
	default:
		return OSTypeOther
	}
}
