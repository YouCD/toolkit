package sysinfo

import (
	"fmt"
	"testing"

	"github.com/youcd/toolkit/sysinfo/types"
)

func Test_getSelinux(t *testing.T) {
	h := new(types.Host)
	getSelinux(h)
	fmt.Println(h.Selinux)
}
