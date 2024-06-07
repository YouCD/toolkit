package sshkey

import (
	"fmt"
	"testing"
)

func TestMakeSSHKeyPair(t *testing.T) {
	got, got1, err := MakeSSHKeyPair()
	if err != nil {
		t.Errorf("MakeSSHKeyPair() error = %v, ", err)
		return
	}
	fmt.Println(got)
	fmt.Println(got1)
}
