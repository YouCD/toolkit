package crane

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"
)

//
//func TestInspect(t *testing.T) {
//	sha256, err := ImageSha256("192.168.110.69/sscra/ms_traefik:1.2.3", "amd64")
//	if err != nil {
//		t.Fatal(err)
//	}
//	t.Log(sha256)
//}
//
//func TestRegistryToRegistry(t *testing.T) {
//	var images = []string{
//		"alpine:latest",
//		"nginx:latest",
//		"ubuntu:latest",
//	}
//
//	for _, image := range images {
//		err := TransferImage("docker.io/library/"+image, "127.0.0.1:5000/"+image, "amd64")
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//
//	t.Log("success")
//}
//
//func TestWriteTarImage(t *testing.T) {
//	// 定义要下载的镜像列表
//
//	startTime := time.Now()
//	imageNames := []string{
//		"192.168.110.69/sscra/ssc_web:dev-231488d5",
//		"192.168.110.69/sscra/ss_weak:dev-2d387d3d",
//	}
//	rename := func(imageName string) string {
//		newRepoTag, err := name.NewTag(path.Base(imageName), name.WithDefaultRegistry("youcdsadasdqq11.com:5000"))
//		if err != nil {
//			t.Error(err)
//			return imageName
//		}
//		return newRepoTag.Name()
//	}
//
//	err := Write2TarballFile("downloaded_images.tar", rename, imageNames...)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	elapsed := time.Since(startTime)
//	fmt.Printf("此次Release耗时: %s\n", elapsed)
//	t.Log("success")
//}

func TestPushTarImage(t *testing.T) {
	now := time.Now()
	msgChan := make(chan Msg)

	go func() {
		defer close(msgChan)
		for msg := range msgChan {
			if msg.Err != nil {
				fmt.Printf("iamge: %s ,状态: %s, err: %v\n", msg.ImageName, msg.State, msg.Err)
			} else {
				fmt.Printf("iamge: %s ,状态: %s\n", msg.ImageName, msg.State)
			}
		}
	}()
	rename := func(imageName string) string {
		base := path.Base(imageName)
		return base
	}
	err := TarballFile2Daemon(os.Args[1], rename, msgChan)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("此次Release耗时: %s\n", time.Since(now))
}

func TestImageTag(t *testing.T) {
	fmt.Println(ImageTag("/tmp/b.tar"))
}
