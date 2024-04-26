package types

import "strings"

type Selinux string

const (
	SelinuxEnforcing  Selinux = "enforcing"
	SelinuxPermissive Selinux = "permissive"
	SelinuxDisabled   Selinux = "disabled"
)

func (s Selinux) String() string {
	return string(s)
}

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
