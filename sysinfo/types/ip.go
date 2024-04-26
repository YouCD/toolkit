package types

import (
	"errors"
	"fmt"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

const (
	IPTypeIPV4 IPType = iota
	IPTypeIPV6
)

var (
	ErrInvalidIP = errors.New("错误的IP地址")
)

type (
	IPType int
	IP     struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
		IPType
	}
)

func (i *IP) String() string {
	return i.Address
}

// FmtIP
//
//	@Description: 格式化输出ip地址，ipv6会加方括号
//	@receiver i
//	@return string
func (i *IP) FmtIP() string {
	switch i.IPType {
	case IPTypeIPV6:
		return fmt.Sprintf("[%s]", i.Address)
	case IPTypeIPV4:
		return i.Address
	default:
		return i.Address
	}
}

// NewIP
//
//	@Description: 将ip 字符串转换为 IP 结构体
//	@param IPStr
//	@return *IP
//	@return error
//
//nolint:gocritic,forbidigo
func NewIP(IPStr string, SSHPort int) (*IP, error) {
	// ip 地址 校验
	ipAddr := ipaddr.NewIPAddressString(IPStr)
	if !ipAddr.IsValid() {
		return nil, ErrInvalidIP
	}
	var ipt IPType
	if ipAddr.IsIPv4() {
		ipt = IPTypeIPV4
	}
	if ipAddr.IsIPv6() {
		ipt = IPTypeIPV6
	}

	return &IP{
		Address: IPStr,
		Port:    SSHPort,
		IPType:  ipt,
	}, nil
}
