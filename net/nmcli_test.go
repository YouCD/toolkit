package net

import (
	"context"
	"testing"
)

func TestNMCli_SetNetWork(t *testing.T) {
	N := &NMCli{}
	addresses := []string{"192.168.111.166/23", "192.168.110.90/23"}
	dns := []string{"8.8.8.8", "223.5.5.5"}
	gateway := "192.168.110.1"
	cni := "ens192"
	err := N.SetNetWork(context.TODO(), addresses, dns, gateway, cni)
	if err != nil {
		t.Errorf("SetNetWork() error = %v", err)
	}
}
