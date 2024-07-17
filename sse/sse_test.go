package sse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"gitlab.firecloud.wan/devops/ops-toolkit/log"
)

func Test_sse(t *testing.T) {
	buffer := bytes.NewBuffer(make([]byte, 1024))
	log.InitBuffer(buffer)
	log.Infof("hello")
	send := 0
	go func() {
		for {
			send++
			log.Infof("hello, 第 %d 次， %s", send, time.Now())
			fmt.Printf("第 %d 次\n", send)
			time.Sleep(time.Second * 2)
		}
	}()
	msgChan := make(chan string)

	defer close(msgChan)
	go func() {
		for {
			//io.C
			//fmt.Println("内容：  ", string(log.LogBuffer))
			line, err := buffer.ReadString('\n')
			if err != nil && !errors.Is(io.EOF, err) {
				log.Errorf("read error: %s", err)
			}
			msgChan <- line

			//n, err := log.LogBuffer.Read(msgChanBy)
			//msgChan <- string(msgChanBy[:n])
			//if err != nil && errors.Is(io.EOF, err) {
			//	if errors.Is(io.EOF, err) {
			//		continue
			//	}
			//	//log.Errorf("read error: %s", err)
			//}

			fmt.Println("Cap  ", buffer.Cap())
			//fmt.Println("内容：  ", log.LogBuffer.String())
			//msgChan <- string(r)
		}
	}()
	Sse(msgChan)
}
