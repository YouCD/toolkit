package net

import (
	"context"
	"testing"
)

func TestNetplan_SetNetWork(t *testing.T) {
	netplan, err2 := NewNetplan("/etc/netplan/00-installer-config.yaml")
	if err2 != nil {
		t.Error(err2)
	}

	err := netplan.SetNetWork(context.Background(), []string{"192.168.111.9/89"}, []string{"8.8.8.8"}, "192.168.110.1", "ens160")
	if err != nil {
		netplan.Rollback(context.Background())
	}

}
