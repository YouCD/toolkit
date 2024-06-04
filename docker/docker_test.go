package docker

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDocker_ContainerLogs(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancelFunc()

	docker, err := NewDocker(ctx)
	if err != nil {
		t.Fatal(err)
	}

	logFilter := func(logs string) bool {
		if strings.Contains(logs, "MySQL init process done. Ready for start up.") {
			fmt.Println("初始化完毕")
			return false
		}
		return false
	}
	err = docker.ContainerLogs(ctx, "mysql", nil, logFilter)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDocker_ContainerImageSha256(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancelFunc()
	d, err := NewDocker(ctx)
	if err != nil {
		t.Fatal(err)
	}
	//sha256, err := d.ContainerImageSha256(ctx, "gitlab")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//fmt.Println(sha256)

	name, err := d.NetworkListByName(ctx, "xxx_default")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(name)

}
