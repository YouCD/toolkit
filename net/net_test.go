package net

import (
	"net"
	"testing"
)

func TestCIDR2IPorNetworkMask(t *testing.T) {
	mask, err := CIDR2IPNet("192.168.1.2/23")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(mask)
}

func TestGetDefaultRouterIP(t *testing.T) {
	routerIP, err := GetDefaultGatewayInterface()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Default router IP: %s", routerIP)

	// 验证返回的IP地址格式是否正确
	if routerIP != "" {
		if ip := net.ParseIP(routerIP); ip == nil {
			t.Errorf("Invalid IP address format: %s", routerIP)
		}
	}
}
