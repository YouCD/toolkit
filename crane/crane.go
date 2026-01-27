package crane

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

var (
	ErrBadName = errors.New("镜像名称错误")
)
var defaultPlatform = v1.Platform{
	Architecture: "amd64",
	OS:           "linux",
}

// DefaultOpt
//
//	@Description: 默认选项
//	@param arch
//	@return []remote.Option
func DefaultOpt(arch string) []remote.Option {
	var platform = defaultPlatform
	platform.Architecture = arch
	opt := make([]remote.Option, 0, 2)
	opt = append(opt,
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithPlatform(platform))
	return opt
}

// ImageSha256
//
//	@Description: 获取镜像的sha256
//	@param dockerURL
//	@param arch
//	@return string
//	@return error
func ImageSha256(dockerURL, arch string) (string, error) {
	// 构建镜像名称对象
	imageRef, err := name.ParseReference(dockerURL)
	if err != nil {
		return "", fmt.Errorf("错误：无法解析镜像名称,err: %w", err)
	}

	// 从远程仓库获取镜像
	img, err := remote.Image(imageRef, DefaultOpt(arch)...)
	if err != nil {
		return "", fmt.Errorf("错误：无法从远程仓库获取镜像,err: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return "", fmt.Errorf("错误：无法获取镜像的manifest,err: %w", err)
	}
	return manifest.Config.Digest.String(), nil
}

// TransferImage
//
//	@Description: 镜像迁移
//	@param srcReg
//	@param dstReg
//	@param arch
//	@return error
func TransferImage(srcReg, dstReg, arch string) error {
	// 解析源镜像的引用
	sourceImageRef, err := name.ParseReference(srcReg)
	if err != nil {
		return fmt.Errorf("错误：无法解析源镜像引用,err: %w", err)
	}

	// 解析目标镜像的引用
	targetImageRef, err := name.ParseReference(dstReg)
	if err != nil {
		return fmt.Errorf("错误：无法解析目标镜像引用,err: %w", err)
	}

	// 从远程仓库下载源镜像
	sourceImg, err := remote.Image(sourceImageRef, DefaultOpt(arch)...)
	if err != nil {
		return fmt.Errorf("错误：无法从远程仓库下载镜像,err: %w", err)
	}

	// 将下载的源镜像上传到目标镜像仓库
	err = remote.Write(targetImageRef, sourceImg, DefaultOpt(arch)...)
	if err != nil {
		return fmt.Errorf("错误：无法将源镜像上传到目标镜像仓库,err: %w", err)
	}
	return nil
}

type renameFunc func(imageName string) string

// Write2TarballFile
//
//	@Description: 将镜像写入 Tarball 文件
//	@param outputFilePath
//	@param rename
//	@param imageNames
//	@return error
func Write2TarballFile(outputFilePath string, renameRegistry renameFunc, imageNames ...string) error {
	var errs []error
	// 定义要保存的输出 tar 文件路径
	imgMap := make(map[string]v1.Image, len(imageNames))
	// 遍历每个镜像名称并下载
	for _, imageName := range imageNames {
		// 解析镜像引用
		imageRef, err := name.ParseReference(imageName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// 从远程仓库下载镜像
		img, err := remote.Image(imageRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// 处理重命名
		newName := imageName
		if renameRegistry != nil {
			newName = renameRegistry(imageName)
		}

		imgMap[newName] = img
	}

	err := crane.MultiSave(imgMap, outputFilePath)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

type state int

const (
	StateStart state = iota + 1
	StateFail
	StateSuccess
)

func (s state) String() string {
	switch s {
	case StateStart:
		return "Start"
	case StateFail:
		return "Fail"
	case StateSuccess:
		return "Success"
	default:
		return "Info"
	}
}

type Msg struct {
	ImageName string
	State     state
	Err       error
}

// TarballFile2Daemon
//
//	@Description: tarballFile 写入到 daemon
//	@param tarballFilePath
//	@param registry
//	@return error
func TarballFile2Daemon(tarballFilePath string, renameFunc renameFunc, msgChan chan Msg) error {
	var errs []error
	//  从文件中读取 清单
	manifest, err := tarball.LoadManifest(pathOpener(tarballFilePath))
	if err != nil {
		return fmt.Errorf("错误：无法从 tarball 文件中读取清单,err: %w", err)
	}
	var wg sync.WaitGroup

	workChan := make(chan struct{}, 10)

	for _, descriptor := range manifest {
		if len(descriptor.RepoTags) == 0 {
			continue
		}
		wg.Add(1)
		workChan <- struct{}{}
		go func(tag string) {
			defer func() {
				<-workChan
				wg.Done()
			}()
			base := path.Base(tag)
			//  将第一个 RepoTags 标签作为镜像的标签
			oldRepoTag, err := name.NewTag(tag)
			if err != nil {
				errs = append(errs, err)
			}

			newRef := oldRepoTag
			//  newRef 名称
			if renameFunc != nil {
				newRef, err = name.NewTag(renameFunc(tag))
				if err != nil {
					errs = append(errs, err)
				}
			}

			// 通过 从文件中读取指定tag镜像
			img, err := tarball.ImageFromPath(tarballFilePath, &oldRepoTag)
			if err != nil {
				errs = append(errs, err)
			}

			msgChan <- Msg{
				ImageName: base,
				State:     StateStart,
				Err:       nil,
			}

			_, err = daemon.Write(newRef, img)
			if err != nil {
				errs = append(errs, err)
				msgChan <- Msg{
					ImageName: base,
					State:     StateFail,
					Err:       err,
				}
			}
			msgChan <- Msg{
				ImageName: base,
				State:     StateSuccess,
				Err:       nil,
			}
		}(descriptor.RepoTags[0])
	}
	wg.Wait()
	return nil
}

func pathOpener(path string) tarball.Opener {
	return func() (io.ReadCloser, error) {
		return os.Open(path)
	}
}

// ImageTag 获取镜像标签
func ImageTag(tarballFilePath string) ([]string, error) {
	//  从文件中读取 清单
	manifest, err := tarball.LoadManifest(pathOpener(tarballFilePath))
	if err != nil {
		return nil, fmt.Errorf("错误：无法从 tarball 文件中读取清单,err: %w", err)
	}

	for _, descriptor := range manifest {
		if len(descriptor.RepoTags) == 0 {
			continue
		}
		return descriptor.RepoTags, nil
	}
	//nolint:err113
	return nil, errors.New("未找到镜像标签")
}

// CountImagesFromTarballFile 获取 tarball 文件中的镜像数量
func CountImagesFromTarballFile(tarballFilePath string) (int, error) {
	manifest, err := tarball.LoadManifest(pathOpener(tarballFilePath))
	if err != nil {
		return 0, fmt.Errorf("错误：无法从 tarball 文件中读取清单,err: %w", err)
	}
	return len(manifest), err
}

// LoadImageFromRemote 从远程仓库下载镜像
func LoadImageFromRemote(ctx context.Context, username, password, image string) error {
	sourceImageRef, err := name.ParseReference(image)
	if err != nil {
		return fmt.Errorf("错误：无法解析源镜像引用,err: %w", ErrBadName)
	}

	// 创建一个自定义的Transport，并禁用证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			//nolint:gosec
			InsecureSkipVerify: true,
		},
	}

	opts := []remote.Option{
		remote.WithAuth(&authn.Basic{Username: username, Password: password}),
		remote.WithTransport(tr),
		remote.WithContext(ctx),
	}
	remoteImg, err := remote.Image(sourceImageRef, opts...)
	if err != nil {
		return fmt.Errorf("错误：无法从远程仓库下载镜像,err: %w", err)
	}
	//nolint:forcetypeassert
	tag := sourceImageRef.(name.Tag)
	_, err = daemon.Write(tag, remoteImg)
	if err != nil {
		return fmt.Errorf("错误：无法将镜像写入本地镜像仓库,err: %w", err)
	}
	return nil
}
