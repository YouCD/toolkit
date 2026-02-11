package net

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/cakturk/go-netstat/netstat"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"github.com/vishvananda/netlink"
)

var (
	ErrNotFoundCNI  = errors.New("not found cni")
	ErrNetplanApply = errors.New("netplan apply error")
)

// GetHostIPByIndex
//
//	@Description: 获取网段的第n个主机位的ip地址
//	@param cidr
//	@return net.IP
func GetHostIPByIndex(cidr string, index int) net.IP {
	addrString := ipaddr.NewIPAddressString(cidr)
	addr := addrString.GetAddress()
	mask := addr.GetNetworkMask()
	networkAddr, _ := addr.Mask(mask)

	subnet := ipaddr.NewIPAddressString(networkAddr.String()).GetAddress().WithoutPrefixLen()
	iterator := subnet.Iterator()
	i := 0
	for next := iterator.Next(); next != nil; next = iterator.Next() {
		i++
		if i == index+1 {
			return next.GetNetIP()
		}
	}
	return nil
}

// CIDR2IPNetworkMask
//
//	@Description: 将CIDR转换成IP地址和掩码
//	@param cidr
//	@return string
//	@return string
func CIDR2IPNetworkMask(cidr string) (string, string, error) {
	ipAddrObjStr := ipaddr.NewIPAddressString(cidr)
	if !ipAddrObjStr.IsValid() {
		//nolint:err113
		return "", "", fmt.Errorf("invalid cidr")
	}

	ipAddrObj := ipAddrObjStr.GetAddress()
	return ipAddrObj.GetNetIPAddr().IP.String(), ipAddrObj.GetNetworkMask().String(), nil
}

var (
	ErrInvalidCIDR = errors.New("invalid cidr")
)

// CIDR2IPNet 将CIDR转换成 netIP
func CIDR2IPNet(cidr string) (string, error) {
	ipAddrObjStr := ipaddr.NewIPAddressString(cidr)
	if !ipAddrObjStr.IsValid() {
		return "", ErrInvalidCIDR
	}

	ipAddrObj := ipAddrObjStr.GetAddress()
	masked, err := ipAddrObj.Mask(ipAddrObj.GetNetworkMask())
	if err != nil {
		return "", fmt.Errorf("ipAddrObj.Mask() : %w", err)
	}
	return masked.String(), err
}

// PhysicsCNIAddress
//
//	@Description: 物理网卡IP地址
//	@return []string
func PhysicsCNIAddress() ([]string, error) {
	var addrs []string
	list, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("netlink.LinkList() : %w", err)
	}

	for _, link := range list {
		// 跳过 MAC地址前缀为02:42 的网卡 libvernet 网卡： virbr0
		if link.Attrs().Name == "lo" || link.Attrs().Name == "veth" ||
			link.Type() == "tuntap" || link.Attrs().HardwareAddr.String() == "" ||
			strings.HasPrefix(link.Attrs().HardwareAddr.String(), "02:42") ||
			strings.HasPrefix(link.Attrs().Name, "virbr") {
			continue
		}
		addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return nil, fmt.Errorf("netlink.AddrList() : %w", err)
		}
		for _, addr := range addrList {
			addrs = append(addrs, addr.IP.String())
		}
	}
	if len(addrs) == 0 {
		return append(addrs, "127.0.0.1"), nil
	}
	return addrs, nil
}

// PhysicsCNI
//
//	@Description: 获取所有物理网卡
//	@return []string
//	@return error
func PhysicsCNI() ([]string, error) {
	cnis := make([]string, 0)
	list, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("netlink.LinkList() : %w", err)
	}

	for _, link := range list {
		// 跳过 MAC地址前缀为02:42 的网卡 libvernet 网卡： virbr0
		if link.Attrs().Name == "lo" || link.Attrs().Name == "veth" ||
			link.Type() == "tuntap" || link.Attrs().HardwareAddr.String() == "" ||
			strings.HasPrefix(link.Attrs().HardwareAddr.String(), "02:42") ||
			strings.HasPrefix(link.Attrs().Name, "virbr") {
			continue
		}
		cnis = append(cnis, link.Attrs().Name)
	}

	return cnis, nil
}

// PhysicsCNIByAddress
//
//	@Description: 查找IP地址对应的网卡
//	@param address
//	@return string
//	@return error
func PhysicsCNIByAddress(address string) (string, error) {
	list, err := netlink.LinkList()
	if err != nil {
		return "", fmt.Errorf("netlink.LinkList() : %w", err)
	}

	for _, link := range list {
		// 跳过 MAC地址前缀为02:42 的网卡 libvernet 网卡： virbr0
		if link.Attrs().Name == "lo" || link.Attrs().Name == "veth" ||
			link.Type() == "tuntap" || link.Attrs().HardwareAddr.String() == "" ||
			strings.HasPrefix(link.Attrs().HardwareAddr.String(), "02:42") ||
			strings.HasPrefix(link.Attrs().Name, "virbr") {
			continue
		}
		addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return "", fmt.Errorf("netlink.AddrList() : %w", err)
		}
		for _, addr := range addrList {
			fmt.Println(addr.IP.String())
			if address == addr.IP.String() {
				return link.Attrs().Name, nil
			}
		}
	}

	return "", ErrNotFoundCNI
}

func SSHPort() int {
	socks, err := netstat.TCPSocks(netstat.NoopFilter)
	if err != nil {
		return 0
	}
	for _, e := range socks {
		if e.Process == nil {
			continue
		}
		if e.Process.Name == "sshd" {
			return int(e.LocalAddr.Port)
		}
	}
	return 0
}

// GetDefaultGatewayInterface 获取默认路由的出口IP
func GetDefaultGatewayInterface() (string, error) {
	// 获取默认路由
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return "", fmt.Errorf("netlink.RouteList() : %w", err)
	}
	var index int
	// 查找默认路由（目标为0.0.0.0/0）
	for _, route := range routes {
		if route.Dst.String() == "0.0.0.0/0" {
			index = route.LinkIndex
			break
		}
	}

	link, err := netlink.LinkByIndex(index)
	if err != nil {
		return "", fmt.Errorf("netlink.LinkByIndex() : %w", err)
	}

	addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return "", fmt.Errorf("netlink.AddrList() : %w", err)
	}
	return addrList[0].IP.String(), nil
}
