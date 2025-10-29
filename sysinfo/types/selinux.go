package types

import "strings"

const (
	SelinuxEnforcing  Selinux = "enforcing"
	SelinuxPermissive Selinux = "permissive"
	SelinuxDisabled   Selinux = "disabled"
)

type Selinux string

func NewSelinux(status string) Selinux {
	status = strings.ToLower(status)
	switch status {
	case "disabled":
		return SelinuxDisabled
	case "permissive":
		return SelinuxPermissive
	default:
		return SelinuxEnforcing
	}
}

func (s Selinux) String() string {
	return string(s)
}
