package bar

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func BenchmarkNewBar(b *testing.B) {
	bar := NewBar("服务检查")
	for n := 0; n < 30; n++ {
		//go func(n int) {
		bar.AddStartBar(strconv.Itoa(n), fmt.Sprintf("检查： %s...", strconv.Itoa(n)))
		bar.SetBarState(strconv.Itoa(n), fmt.Sprintf("%25s：OK", strconv.Itoa(n)), StateSuccess)
		//}(n)
	}
}

func TestNewBar(t *testing.T) {
	fmt.Println("sssssssssssss")
	//b := NewBar("服务检查")
	//services := []string{"aaa", "bbb", "ccc", "ddd", "eee", "ffff", "gggg", "hhhh"}
	//for _, svc := range services {
	//	b.AddStartBar(svc, fmt.Sprintf("检查： %s...", svc))
	//	b.SetBarState(svc, fmt.Sprintf("%25s：OK", svc), StateSuccess)
	//	//if newDocker.ContainerCheckFuncIsHealthOrRunning(svc) {
	//	//
	//	//} else {
	//	//	if err := newDocker.ContainerStart(svc); err != nil {
	//	//		log.Error(err.Error())
	//	//	}
	//	//	b.SetBarState(svc, fmt.Sprintf("%25s：%s", svc, newDocker.ContainerState(svc)), bar.StateFail)
	//	//}
	//}
	bar := NewBar("服务检查")
	for n := 0; n < 30; n++ {
		//go func(n int) {
		bar.AddStartBar(strconv.Itoa(n), fmt.Sprintf("检查： %s...", strconv.Itoa(n)))
		time.Sleep(time.Millisecond * 100)
		bar.SetBarState(strconv.Itoa(n), fmt.Sprintf("%25s：OK", strconv.Itoa(n)), StateSuccess)
		//}(n)
	}
	bar.Stop()
}
