package systemd

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewSystemd(t *testing.T) {
	var (
		dbusStatusCh  = make(chan ConnectionState)
		msgChan       = make(chan Msg)
		unitCheckChan = make(chan UnitCheck)
	)
	go func() {
		for {
			select {
			case state := <-dbusStatusCh:
				fmt.Println("dbusStatus        ", "连接状态     ", state.Connected, "totalAttempts      ", state.TotalAttempts, "err           ", state.Err)
			case msg := <-msgChan:
				fmt.Println("unit               ", msg.UnitName, "msgStr           ", msg.MsgStr)
			case unitCheck := <-unitCheckChan:
				fmt.Println("unitCheck         ", "unitName      ", unitCheck.UnitName, "checkCount      ", unitCheck.CheckCount)
			}
		}
	}()
	ctx := context.TODO()

	systemCtl, err := NewSystemd(ctx, nil, nil, nil)
	if err != nil {
		os.Exit(1)
	}

	err = systemCtl.UnitRestart(ctx, "docker.service")
	if err != nil {
		fmt.Println("err               ", err)
	}
	time.Sleep(2 * time.Second)
	systemCtl, err = NewSystemd(ctx, dbusStatusCh, msgChan, unitCheckChan)
	if err != nil {
		os.Exit(1)
	}
	err = systemCtl.UnitStartWithEnable(ctx, "docker.service")
	if err != nil {
		fmt.Println("err               ", err)
	}

}
