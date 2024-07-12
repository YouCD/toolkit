package net

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"gitlab.firecloud.wan/devops/ops-toolkit/file"
	"gopkg.in/yaml.v3"
)

type Nameservers struct {
	Addresses []string `yaml:"addresses"`
}
type Ethernet struct {
	Dhcp4       bool        `yaml:"dhcp4"`
	Dhcp6       bool        `yaml:"dhcp6"`
	Addresses   []string    `yaml:"addresses"`
	Nameservers Nameservers `yaml:"nameservers"`
	Gateway4    string      `yaml:"gateway4"`
}
type Route struct {
	To  string `yaml:"to"`
	Via string `yaml:"via"`
}

type Bridge struct {
	Interfaces  []string `yaml:"interfaces"`
	Addresses   []string `yaml:"addresses"`
	Routes      []Route  `yaml:"routes"`
	Nameservers struct {
		Addresses []string `yaml:"addresses"`
	} `yaml:"nameservers"`
	Parameters struct {
		Stp          bool `yaml:"stp"`
		ForwardDelay int  `yaml:"forward-delay"` //nolint
	} `yaml:"parameters"`
	Dhcp4 string `yaml:"dhcp4,omitempty"`
	Dhcp6 string `yaml:"dhcp6,omitempty"`
}
type Network struct {
	Renderer  string              `yaml:"renderer,omitempty"`
	Ethernets map[string]Ethernet `yaml:"ethernets"`
	Bridges   map[string]Bridge   `yaml:"bridges,omitempty"`
	Version   int                 `yaml:"version"`
}
type NetworkObj struct {
	Network `yaml:"network"`
}

type Netplan struct {
	NetworkObj     *NetworkObj `yaml:"network"`
	ConfigFile     string
	ConfigFileData []byte
}

func NewNetplan(configFile string) (*Netplan, error) {
	netplan := new(Netplan)
	netplan.ConfigFile = configFile
	if err := netplan.ParserConfig(); err != nil {
		return nil, err
	}

	return netplan, nil
}

// Rollback
//
//	@Description: 回滚
//	@receiver n
//	@return error
func (n *Netplan) Rollback() error {
	timeDirName := time.Now().Format("20060102")
	backupFile := n.ConfigFile + "." + timeDirName
	readFile, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("read backupFile error: %w", err)
	}
	if err := os.WriteFile(n.ConfigFile, readFile, 0644); err != nil {
		return fmt.Errorf("write file error: %w", err)
	}
	return n.apply()
}

// GetCNI
//
//	@Description: 获取指定IP的cni
//	@receiver n
//	@param addresses
//	@return string
//	@return error
func (n *Netplan) GetCNI(addresses string) (string, error) {
	cni, err := PhysicsCNIByAddress(addresses)
	if err != nil {
		return "", fmt.Errorf("PhysicsCNIByAddress: %w", err)
	}
	return cni, err
}

// ParserConfig
//
//	@Description: 解析配置文件
//	@receiver n
//	@return error
func (n *Netplan) ParserConfig() error {
	fileData, err := os.ReadFile(n.ConfigFile)
	if err != nil {
		return fmt.Errorf("read fileData error: %w", err)
	}
	var networkData NetworkObj
	if err := yaml.Unmarshal(fileData, &networkData); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}
	n.NetworkObj = &networkData
	n.ConfigFileData = fileData
	return nil
}

// SetNetWork
//
//	@Description: 设置网络信息
//	@receiver n
//	@param addresses []string
//	@param dns
//	@param gateway
//	@param cni
//	@return error
func (n *Netplan) SetNetWork(addresses, dns []string, gateway, cni string) error {
	// 1. 修改网卡配置
	ethernet, ok := n.NetworkObj.Ethernets[cni]
	if !ok {
		return fmt.Errorf("ethernet: %s,err: %w", cni, ErrNotFoundCNI)
	}
	ethernet.Addresses = addresses
	ethernet.Gateway4 = gateway
	ethernet.Nameservers.Addresses = dns
	n.NetworkObj.Ethernets[cni] = ethernet

	// 2. 备份配置文件
	timeDirName := time.Now().Format("20060102")
	backupFile := n.ConfigFile + "." + timeDirName
	if !file.Exists(backupFile) {
		err := os.WriteFile(backupFile, n.ConfigFileData, 0644)
		if err != nil {
			return fmt.Errorf("write file , file: %s ,error: %w", backupFile, err)
		}
	}

	// 3. 写入配置文件
	out, err := yaml.Marshal(n.NetworkObj)
	if err != nil {
		return fmt.Errorf("yaml.marshal error: %w", err)
	}
	if err := os.WriteFile(n.ConfigFile, out, 0644); err != nil {
		return fmt.Errorf("write file , file: %s ,error: %w", n.ConfigFile, err)
	}
	// 4. 应用配置
	return n.apply()
}

// apply
//
//	@Description: 应用
//	@receiver n
//	@return error
func (n *Netplan) apply() error {
	cmdStr := "netplan apply"
	return bash(cmdStr)
}

func bash(cmdStr string) error {
	command := exec.Command("bash", "-c", cmdStr)
	command.Env = append(command.Environ(), "LANG=en_US.utf8", "LANGUAGE=en_US.utf8")
	output, err := command.CombinedOutput()
	if err != nil {
		//nolint:errorlint
		if ins, ok := err.(*exec.ExitError); ok {
			out := string(output)
			exitCode := ins.ExitCode()
			return fmt.Errorf("out: %s, exit.code: %d,err %w", out, exitCode, ErrNetplanApply)
		}
		return fmt.Errorf("out: %s,err:%w", string(output), ErrNetplanApply)
	}
	return nil
}
