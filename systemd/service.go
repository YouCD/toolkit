package systemd

import "strings"

type ServiceStatus string

const (
	ServiceStatusRunning    ServiceStatus = "running"
	ServiceStatusStopped    ServiceStatus = "stopped"
	ServiceStatusFailed     ServiceStatus = "failed"
	ServiceStatusStatic     ServiceStatus = "static"
	ServiceStatusActivating ServiceStatus = "activating"
	ServiceStatusActivate   ServiceStatus = "active"
	ServiceStatusEnabled    ServiceStatus = "enabled"
	ServiceStatusDisabled   ServiceStatus = "disabled"
	ServiceStatusMasked     ServiceStatus = "masked"
	ServiceStatusOther      ServiceStatus = "other"
	ServiceStatusUnknown    ServiceStatus = "unknown"
)

type Service struct {
	Name          string
	CurrentStatus ServiceStatus
	NeedStatus    ServiceStatus
}

func (s Service) String() string {
	return string(s.CurrentStatus)
}
func Str2ServiceStatus(str string) ServiceStatus {
	switch strings.ToLower(str) {
	case "running":
		return ServiceStatusRunning
	case "stopped":
		return ServiceStatusStopped
	case "failed":
		return ServiceStatusFailed
	case "static":
		return ServiceStatusStatic
	case "activating":
		return ServiceStatusActivating
	case "active":
		return ServiceStatusActivate
	case "enabled":
		return ServiceStatusEnabled
	case "disabled":
		return ServiceStatusDisabled
	case "masked":
		return ServiceStatusMasked
	case "other":
		return ServiceStatusOther
	default:
		return ServiceStatusUnknown
	}
}
