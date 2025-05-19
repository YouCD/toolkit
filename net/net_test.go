package net

import "testing"

func TestCIDR2IPorNetworkMask(t *testing.T) {
	mask, err := CIDR2IPNet("192.168.1.2/23")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(mask)
}
