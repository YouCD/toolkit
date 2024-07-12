package net

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v3/host"
	"gitlab.firecloud.wan/devops/ops-toolkit/sysinfo/types"
)

var (
	ErrNotFoundNetplanConfig = errors.New("not found netplan config")
)

type NetworkManager interface {
	SetNetWork(addresses, dns []string, gateway, cni string) error
	GetCNI(addresses string) (string, error)
	Rollback() error
}

func NewNetworkManager() (NetworkManager, error) {
	h := new(types.Host)
	// 系统信息
	info, err := host.Info()
	if err == nil {
		h.HostInfo = info
	}
	if h.Platform() == types.OSPlatformUbuntu {
		var configFile string
		err = filepath.Walk("/etc/netplan", func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("filepath.Walk(), err: %w", err)
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".yaml") {
				configFile = path
				return nil
			}
			return ErrNotFoundNetplanConfig
		})
		if err != nil {
			return nil, fmt.Errorf("ubuntu系统获取netplan配置失败, err: %w", err)
		}
		return NewNetplan(configFile)
	}
	return NewNMcli(), nil
}
