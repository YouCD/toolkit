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
