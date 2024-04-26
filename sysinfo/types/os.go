package types

type OSPlatform string

const (
	OSPlatformCentos       OSPlatform = "centos"
	OSPlatformUbuntu       OSPlatform = "ubuntu"
	OSPlatformBigCloud     OSPlatform = "bigcloud"
	OSPlatformAnolis       OSPlatform = "anolis"
	OSPlatformOpeneuler    OSPlatform = "openeuler"
	OSPlatformUOS          OSPlatform = "uos"
	OSPlatformOpensuseLeap OSPlatform = "opensuse-leap"
	OSPlatformRocky        OSPlatform = "rocky"
	OSPlatformOther        OSPlatform = "other"
)

func (o OSPlatform) String() string {
	return string(o)
}
