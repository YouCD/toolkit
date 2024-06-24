package net

import (
	"fmt"
	"strings"
)

type NMCli struct {
}

func NewNMcli() *NMCli {
	return &NMCli{}
}

func (n *NMCli) SetNetWork(addresses, dns []string, gateway, cni string) error {
	//  nmcli connection modify br0 ipv4.gateway 192.168.104.88
	addressesStr := strings.Join(addresses, ",")
	dnsStr := strings.Join(dns, ",")
	cmdStr := fmt.Sprintf("nmcli connection modify %s ipv4.addresses %s ipv4.gateway %s ipv4.dns %s ifname %s", cni, addressesStr, gateway, dnsStr, cni)
	err := bash(cmdStr)
	if err != nil {
		return fmt.Errorf("bash: %w", err)
	}
	err = bash("nmcli connection up " + cni)
	if err != nil {
		return fmt.Errorf("bash: %w", err)
	}
	return nil
}

func (n *NMCli) GetCNI(addresses string) (string, error) {
	cni, err := PhysicsCNIByAddress(addresses)
	if err != nil {
		return "", fmt.Errorf("PhysicsCNIByAddress: %w", err)
	}
	return cni, err
}

func (n *NMCli) Rollback() error {
	return nil
}
