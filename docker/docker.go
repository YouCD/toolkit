package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	types2 "github.com/compose-spec/compose-go/v2/types"
	"github.com/distribution/reference"
	"github.com/docker/buildx/driver"
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"github.com/docker/compose/v2/pkg/progress"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/libnetwork/ipamapi"
	"github.com/docker/docker/registry"
	"github.com/docker/go-connections/nat"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/youcd/toolkit/net"
)

var (
	ErrCheckTimeout                 = errors.New("check timeout")
	ErrDockerComposeProjectNotFound = errors.New("docker Compose project not found")
	ErrDockerNetworkPollExist       = errors.New("DockerIP网络池已分配")
	ErrNetworkExist                 = errors.New("docker network exist")
	ErrNetworkNotExist              = errors.New("docker network not exist")
	ErrNoYaml                       = errors.New("没有找到ymal文件")
	ErrSubnetGateway                = errors.New("docker.subnet.gateway 未找到")
	ErrSubnetCIDR                   = errors.New("docker.subnet.cidr 未找到")
)

var (
	d *Docker
)

type Docker struct {
	ComposeService  api.Service
	DockerCLIClient command.Cli
	EnvFiles        []string
}

func NewDocker(ctx context.Context, envFiles ...string) (*Docker, error) {
	// 关闭进度条
	progress.Mode = progress.ModeQuiet
	// 创建docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("new docker dockerClient error: %w", err)
	}
	// 检查docker是否可用
	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("new docker dockerClient,err: %w", err)
	}

	// 创建docker cli  client
	dockerCli, err := command.NewDockerCli(command.WithAPIClient(dockerClient))
	if err != nil {
		panic(err)
	}
	// 初始化 docker cli  client
	err = dockerCli.Initialize(cliflags.NewClientOptions())
	if err != nil {
		return nil, fmt.Errorf("new docker cli client error: %w", err)
	}
	// 创建compose service
	service := compose.NewComposeService(dockerCli)

	d = &Docker{
		ComposeService:  service,
		DockerCLIClient: dockerCli,
		EnvFiles:        envFiles,
	}
	return d, nil
}

func (d *Docker) Close() {
	_ = d.DockerCLIClient.Client().Close()
}

// watchContainerCallback 回调函数
type watchContainerCallback func(ctx context.Context, containerName string, rowData events.Message) error

// WatchContainerCreate
//
//	@Description: watch 容器的事件
//	@receiver d
//	@param watch
func (d *Docker) WatchContainerCreate(ctx context.Context, action events.Action, watch watchContainerCallback) error {
	evs, errs := d.DockerCLIClient.Client().Events(ctx, events.ListOptions{
		Filters: filters.NewArgs(filters.Arg("type", string(events.ContainerEventType))), // 搞一个事件过滤器
	})

	for {
		select {
		case event := <-evs:
			// 搞到事件
			// event.Action  exec_start exec_die start
			var containerName string
			// 优先使用compose服务名
			s, ok := event.Actor.Attributes["com.docker.compose.service"]
			if ok {
				containerName = s
			} else {
				containerName = event.Actor.Attributes["name"]
			}

			if event.Action == action && watch != nil {
				err := watch(ctx, containerName, event)
				if err != nil {
					return err
				}
			}
		case err := <-errs:
			return err
		}
	}
}

type ContainerCheckFunc func(ctx context.Context, containerName string) bool

// ContainerCheckFuncIsHealth
//
//	@Description: 容器是否健康
//	@receiver d
//	@param container
//	@return bool
func (d *Docker) ContainerCheckFuncIsHealth(ctx context.Context, container string) bool {
	inspect, err := d.Inspect(ctx, container)
	if err != nil {
		return false
	}
	if inspect.State.Health != nil && inspect.State.Health.Status == "healthy" {
		return true
	}
	return false
}

// ContainerCheckFuncIsHealthOrRunning
//
//	@Description: 容器处于健康或者运行状态
//	@receiver d
//	@param container
//	@return bool
func (d *Docker) ContainerCheckFuncIsHealthOrRunning(ctx context.Context, container string) bool {
	// 优先使用health属性
	if d.ContainerHasHealthAttributes(ctx, container) {
		return d.ContainerCheckFuncIsHealth(ctx, container)
	}
	return d.ContainerCheckFuncIsRunning(ctx, container)
}

// ContainerHasHealthAttributes
//
//	@Description: 容器是否有健康属性
//	@receiver d
//	@param container
//	@return bool
func (d *Docker) ContainerHasHealthAttributes(ctx context.Context, container string) bool {
	inspect, err := d.Inspect(ctx, container)
	if err != nil {
		return false
	}
	return inspect.State.Health != nil
}

// ContainerCheckFuncIsRunning
//
//	@Description:容器是否运行
//	@receiver d
//	@param container
//	@return bool
func (d *Docker) ContainerCheckFuncIsRunning(ctx context.Context, container string) bool {
	inspect, err := d.Inspect(ctx, container)
	if err != nil {
		return false
	}
	if inspect.State.Status == "running" {
		return true
	}
	return false
}

// ContainerCheck
//
//	@Description: 检查容器状态
//	@receiver d
//	@param service
//	@param checkFunc
//	@return error
func (d *Docker) ContainerCheck(ctx context.Context, service string, checkFunc ContainerCheckFunc) error {
	ticker := time.NewTicker(time.Second * 300)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			return ErrCheckTimeout
		default:
			if checkFunc(ctx, service) {
				return nil
			}
			// 启动一把
			_ = d.ContainerStart(ctx, service)
			time.Sleep(5 * time.Second)
		}
	}
}

// ContainerState
//
//	@Description:容器当前状态
//	@receiver d
//	@param container
//	@return string
func (d *Docker) ContainerState(ctx context.Context, container string) string {
	inspect, err := d.Inspect(ctx, container)
	if err != nil {
		return "Not Funded"
	}
	//  优先返回健康状态
	if inspect.State.Health != nil {
		return inspect.State.Health.Status
	}

	return inspect.State.Status
}

// ContainerStart
//
//	@Description:启动容器
//	@receiver d
//	@param container
//	@return error
func (d *Docker) ContainerStart(ctx context.Context, containers ...string) error {
	var errs []error
	for _, containerName := range containers {
		inspect, err := d.Inspect(ctx, containerName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = d.DockerCLIClient.Client().ContainerStart(ctx, inspect.ID, container.StartOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// ContainerList
//
//	@Description: 获取容器列表
//	@receiver d
//	@param ctx
//	@return []string
//	@return error
func (d *Docker) ContainerList(ctx context.Context) ([]types.Container, error) {
	//nolint:wrapcheck
	return d.DockerCLIClient.Client().ContainerList(ctx, container.ListOptions{})
}

// ContainerStop
//
//	@Description: 停止容器
//	@receiver d
//	@param containerName
//	@return error
func (d *Docker) ContainerStop(ctx context.Context, containers ...string) error {
	var errs []error
	for _, containerName := range containers {
		inspect, err := d.Inspect(ctx, containerName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = d.DockerCLIClient.Client().ContainerStop(ctx, inspect.ID, container.StopOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// FindSVCFromYamlFile
//
//	@Description: 获取svc 所在的 Yaml 文件
//	@receiver d
//	@param ctx
//	@param service
//	@param project
//	@return string
//	@return []string
func (d *Docker) FindSVCFromYamlFile(ctx context.Context, service, project string) (string, []string, error) {
	stack, err := d.ComposeListByName(ctx, project)
	if err != nil {
		return "", nil, err
	}
	yamlFiles := strings.Split(stack.ConfigFiles, ",")
	var yamlFile string
	for _, file := range yamlFiles {
		if _, ok, _ := d.svcInComposeFile(ctx, file, service); ok {
			yamlFile = file
		}
	}

	if yamlFile == "" {
		return "", nil, err
	}
	return yamlFile, yamlFiles, nil
}

// ContainerUpdateImage
//
//	@Description: 更新容器镜像
//	@receiver d
//	@param containerName
//	@param image
//	@return error
func (d *Docker) ContainerUpdateImage(ctx context.Context, containerName string, image string, pull bool) error {
	// 获取容器配置
	inspectJSON, err := d.Inspect(ctx, containerName)
	if err != nil {
		return err
	}
	if pull {
		// 拉取新镜像
		err = d.ImagePull(ctx, image, nil)
		if err != nil {
			return fmt.Errorf("ImagePull() error: %w", err)
		}
	}

	// 删除旧容器
	err = d.ContainerRemove(ctx, containerName)
	if err != nil {
		return err
	}

	// 更新镜像字段
	inspectJSON.Config.Image = image

	// 构建新的网络配置：只保留网络名，不复制 IP/MAC
	EndpointsConfig := make(map[string]*network.EndpointSettings)
	for netName := range inspectJSON.NetworkSettings.Networks {
		EndpointsConfig[netName] = &network.EndpointSettings{}
	}
	networkingConfig := &network.NetworkingConfig{EndpointsConfig: EndpointsConfig}

	// 创建并启动容器
	for {
		resp, err := d.ContainerCreate(ctx, containerName, inspectJSON, networkingConfig)
		if err != nil {
			return err
		}

		err = d.DockerCLIClient.Client().ContainerStart(ctx, resp.ID, container.StartOptions{})
		if err != nil {
			// 某些容器无用户配置
			if strings.Contains(err.Error(), "no matching entries in passwd file") {
				_ = d.ContainerRemove(ctx, containerName)
				inspectJSON.Config.User = ""
				continue
			}
			return fmt.Errorf("ContainerStart() error: %w", err)
		}
		break
	}

	return nil
}

// ContainerCreate
//
//	@Description: 创建容器
//	@receiver d
//	@param containerName
//	@param inspectJSON
//	@param networkingConfig
//	@return container.CreateResponse
//	@return error
func (d *Docker) ContainerCreate(ctx context.Context, containerName string, inspectJSON *container.InspectResponse, networkingConfig *network.NetworkingConfig) (container.CreateResponse, error) {
	resp, err := d.DockerCLIClient.Client().ContainerCreate(ctx,
		inspectJSON.Config,
		inspectJSON.HostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("ContainerCreatey() error: %w", err)
	}
	return resp, nil
}

// ContainerIsExits
//
//	@Description:容器是否创建
//	@receiver d
//	@param container
//	@return error
func (d *Docker) ContainerIsExits(ctx context.Context, containerName string) bool {
	inspect, err := d.DockerCLIClient.Client().ContainerList(ctx, container.ListOptions{All: true, Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: containerName})})
	if err != nil {
		return false
	}
	if len(inspect) == 0 {
		return false
	}
	return true
}

// Inspect
//
//	@Description: docker inspect 容器
//	@receiver d
//	@param containerName
//	@return *container.InspectResponse
//	@return error
func (d *Docker) Inspect(ctx context.Context, containerName string) (*container.InspectResponse, error) {
	inspect, err := d.DockerCLIClient.Client().ContainerInspect(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("%s :inspect error,err: %w", containerName, err)
	}
	return &inspect, nil
}

// ContainerIP 获取容器IP
func (d *Docker) ContainerIP(ctx context.Context, containerName, subnetName string) (string, error) {
	inspect, err := d.Inspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("%s :inspect error,err: %w", containerName, err)
	}
	if ins, ok := inspect.NetworkSettings.Networks[subnetName]; ok {
		return ins.IPAddress, nil
	}
	return "", ErrSubnetCIDR
}

// ContainerRemove
//
//	@Description:删除容器
//	@receiver d
//	@param containerName
//	@return error
func (d *Docker) ContainerRemove(ctx context.Context, containerName string) error {
	// 如果没有 containerName 直接返回
	if !d.ContainerIsExits(ctx, containerName) {
		return nil
	}

	containerJSON, err := d.Inspect(ctx, containerName)
	if err != nil {
		return fmt.Errorf("inspect() error: %w", err)
	}

	err = d.DockerCLIClient.Client().ContainerRemove(ctx, containerJSON.ID, container.RemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("%s :ContainerRemove error, err: %w", containerName, err)
	}

	return nil
}

// CreateRegistry
//
//	@Description:创建 Registry 容器，如果已经存在则删除再创建
//	@receiver d
//	@param image
//	@param repoPath
//	@return error
func (d *Docker) CreateRegistry(ctx context.Context, imageName, repoPath, hostPort string) error {
	registryName := "registry"
	if d.ContainerIsExits(ctx, registryName) {
		err := d.ContainerRemove(ctx, registryName)
		if err != nil {
			return fmt.Errorf("registry容器删除失败 error: %w", err)
		}
	}

	if imageName != "" {
		// 导入 registry 镜像
		err := d.ImageLoadFromFile(ctx, imageName)
		if err != nil {
			return fmt.Errorf("load镜像 error: %w", err)
		}
	} else {
		imageResp, err := d.DockerCLIClient.Client().ImagePull(ctx, registryName, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("pull registry image error: %w", err)
		}
		defer func() {
			_ = imageResp.Close()
		}()
		var buf bytes.Buffer
		_, err = io.Copy(&buf, imageResp)
		if err != nil {
			return fmt.Errorf("copy image error: %w", err)
		}
	}

	err := os.MkdirAll(repoPath, 0755)
	if err != nil {
		return fmt.Errorf("创建挂载目录 error: %w", err)
	}
	// 文件挂载
	m := make([]mount.Mount, 0, 1)
	m = append(m, mount.Mount{Type: mount.TypeBind, Source: repoPath, Target: "/var/lib/registry"})

	exports := make(nat.PortSet)
	netPort := make(nat.PortMap)

	// 网络端口映射
	natPort, _ := nat.NewPort("tcp", "5000")
	exports[natPort] = struct{}{}
	dstPort := "5000"
	if hostPort == "" {
		hostPort = dstPort
	}
	portList := make([]nat.PortBinding, 0, 1)
	portList = append(portList, nat.PortBinding{HostIP: "0.0.0.0", HostPort: hostPort})
	netPort[natPort] = portList

	// 创建容器
	resp, err := d.DockerCLIClient.Client().ContainerCreate(ctx,
		&container.Config{
			Image:        registryName,
			ExposedPorts: exports,
			//Cmd:          cmd,
			Tty: false,
			// WorkingDir:   workDir,
		},
		&container.HostConfig{
			PortBindings: netPort,
			Mounts:       m,
			AutoRemove:   true, // 自动删除
		}, nil, nil, registryName)

	if err != nil {
		return fmt.Errorf("registry创建 error: %w", err)
	}

	err = d.DockerCLIClient.Client().ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("启动registry error: %w", err)
	}
	return nil
}

// ImageLoadFromFile
//
//	@Description: docker load
//	@receiver d
//	@param imagePath
//	@return error
func (d *Docker) ImageLoadFromFile(ctx context.Context, imagePath string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("加载镜像 error: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	return d.imageLoadFromIOReader(ctx, file)
}

// ImagePull
//
//	@Description: 镜像拉取
//	@receiver d
//	@param regImage
//	@param newTagFunc
//	@return error
func (d *Docker) ImagePull(ctx context.Context, regImage string, newTagFunc func(string) string) error {
	// 获取认证信息 ~/.docker/config.json
	authConfigs, err := d.matchAuthConfig(regImage)
	if err != nil {
		return err
	}

	imageResp, err := d.DockerCLIClient.Client().ImagePull(ctx, regImage, image.PullOptions{RegistryAuth: authConfigs})
	if err != nil {
		return fmt.Errorf("pull镜像:%s error: %w", regImage, err)
	}
	defer func() {
		_ = imageResp.Close()
	}()
	buf := bytes.Buffer{}
	_, err = io.Copy(&buf, imageResp)
	if err != nil {
		return fmt.Errorf("copy image error: %w", err)
	}
	if newTagFunc != nil {
		newName := newTagFunc(regImage)
		err = d.DockerCLIClient.Client().ImageTag(ctx, regImage, newName)
		if err != nil {
			return fmt.Errorf("tag镜像:%s error: %w", newName, err)
		}
		_, err = d.DockerCLIClient.Client().ImageRemove(ctx, regImage, image.RemoveOptions{})
		if err != nil {
			return fmt.Errorf("remove镜像:%s error: %w", regImage, err)
		}
	}
	return nil
}

// NetworkCreate
//
//	@Description:创建 docker net
//	@receiver d
//	@param netName
//	@param ipCIDR
//	@return error
func (d *Docker) NetworkCreate(ctx context.Context, netName, ipCIDR string) error {
	list, err := d.DockerCLIClient.Client().NetworkList(ctx, network.ListOptions{Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: netName})})
	if err != nil {
		return fmt.Errorf("list net %s,err:%w", netName, err)
	}
	if len(list) > 0 {
		return ErrNetworkExist
	}

	options := network.CreateOptions{
		// CheckDuplicate: false,
		Driver: "bridge",
		IPAM: &network.IPAM{
			Driver:  "default",
			Options: nil,
			Config: []network.IPAMConfig{{
				Subnet:  ipCIDR,
				Gateway: net.GetHostIPByIndex(ipCIDR, 1).String(),
			}},
		},
	}

	_, err = d.DockerCLIClient.Client().NetworkCreate(ctx, netName, options)
	if err != nil {
		if errors.Is(err, ipamapi.ErrPoolOverlap) {
			return ErrDockerNetworkPollExist
		}
		return fmt.Errorf("create net %s,err:%s", netName, err.Error()) //nolint
	}

	return nil
}

// NetworkListByName
//
//	@Description: 通过名称获取 Network
//	@receiver d
//	@param ctx
//	@param netName
//	@return []types.NetworkResource
//	@return error
func (d *Docker) NetworkListByName(ctx context.Context, netName string) (*network.Summary, error) {
	list, err := d.DockerCLIClient.Client().NetworkList(ctx, network.ListOptions{Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: netName})})
	if err != nil {
		return nil, fmt.Errorf("list Network,err:%w", err)
	}
	if len(list) == 0 {
		return nil, ErrNetworkNotExist
	}
	return &list[0], nil
}

// ComposeServiceUp  docker-compose up
//
//	@Description:
//	@receiver d
//	@param p
//	@param recreateMod api.RecreateNever
//	@return error
func (d *Docker) ComposeServiceUp(ctx context.Context, p *types2.Project, recreateMod string) error {
	upOpts := api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans:        true,
			Recreate:             recreateMod,
			RecreateDependencies: recreateMod,
			Inherit:              true,
			QuietPull:            true,
		},
		Start: api.StartOptions{
			Project:     p,
			OnExit:      api.CascadeStop,
			Wait:        true,
			WaitTimeout: time.Second * 3000,
		},
	}
	err := d.ComposeService.Up(ctx, p, upOpts)
	if err != nil {
		return fmt.Errorf("compose up, error: %w", err)
	}
	return nil
}

// ComposeServiceRestart
//
//	@Description: 服务重启
//	@receiver d
//	@param p
//	@return error
func (d *Docker) ComposeServiceRestart(ctx context.Context, p *types2.Project) error {
	Opts := api.RestartOptions{
		Project: p,
		//Timeout:  nil,
		//Services: nil,
		//NoDeps:   false,
	}

	err := d.ComposeService.Restart(ctx, p.Name, Opts)
	if err != nil {
		return fmt.Errorf("compose restart, error: %w", err)
	}
	return nil
}

// ComposeFilterSvcFromProject
//
//	@Description: 从Project中过滤出匹配的Services
//	@receiver d
//	@param svcName
//	@param project
//	@return *types2.Project
func (d *Docker) ComposeFilterSvcFromProject(svcName []string, project *types2.Project) *types2.Project {
	result := new(types2.Project)
	result.Networks = project.Networks
	delete(result.Networks, "default")
	result.Extensions = project.Extensions
	result.Name = project.Name
	services := make(types2.Services)
	for _, service := range project.Services {
		if slice.Contain(svcName, service.Name) {
			services[service.Name] = service
		}
	}
	result.Services = services
	return result
}

// ComposeList
//
//	@Description: Compose List
//	@receiver d
//	@return []api.Stack
//	@return error
func (d *Docker) ComposeList(ctx context.Context) ([]api.Stack, error) {
	opts := api.ListOptions{
		All: true,
	}
	list, err := d.ComposeService.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("compose ls, error: %w", err)
	}
	return list, nil
}

// ComposeListByName
//
//	@Description: 按projectName列出compose项目
//	@receiver d
//	@param projectName
//	@return *api.Stack
//	@return error
func (d *Docker) ComposeListByName(ctx context.Context, projectName string) (*api.Stack, error) {
	opts := api.ListOptions{
		All: true,
	}
	list, err := d.ComposeService.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("compose ls, error: %w", err)
	}

	for _, stack := range list {
		if stack.Name == projectName {
			return &stack, nil
		}
	}

	return nil, ErrDockerComposeProjectNotFound
}

// encodedAuth
//
//	@Description: 从配置文件解码认证信息
//	@param ref
//	@param configFile
//	@return string
//	@return error
func encodedAuth(ref reference.Named, configFile driver.Auth) (string, error) {
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", fmt.Errorf("parse repository info, error: %w", err)
	}

	key := registry.GetAuthConfigKey(repoInfo.Index)
	authConfig, err := configFile.GetAuthConfig(key)
	if err != nil {
		return "", fmt.Errorf("get auth config, error: %w", err)
	}

	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("marshal auth config, error: %w", err)
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

// ComposeYamlRead
//
//	@Description: 从yaml中读取compose项目
//	@param file
//	@return *types2.Project
//	@return error
func ComposeYamlRead(ctx context.Context, file string, envFiles ...string) (*types2.Project, error) {
	opts, err := cli.NewProjectOptions(
		[]string{file},
		//  关闭一致性校验
		cli.WithConsistency(false),
		cli.WithEnvFiles(envFiles...),
		cli.WithDotEnv,
		cli.WithLoadOptions(loader.WithSkipValidation),
	)
	if err != nil {
		return nil, fmt.Errorf("NewProjectOptions() err:%w", err)
	}
	//nolint:wrapcheck
	return opts.LoadProject(ctx)
}

type LogFilter func(logs string) bool

func (d *Docker) ContainerLogs(ctx context.Context, containerName string, options *container.LogsOptions, filter LogFilter) error {
	// 默认选项
	defaultOptions := &container.LogsOptions{ShowStdout: true, Follow: true}
	if options != nil {
		defaultOptions = options
	}
	reader, err := d.DockerCLIClient.Client().ContainerLogs(ctx, containerName, *defaultOptions)
	if err != nil {
		return fmt.Errorf("container logs, error: %w", err)
	}
	hdr := make([]byte, 8)
	for {
		_, err = reader.Read(hdr)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// 如果是 follow 模式，可以继续等待
				if defaultOptions.Follow {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				// 否则正常退出
				return nil
			}
			return fmt.Errorf("container logs, Read head error: %w", err)
		}

		var buffer bytes.Buffer
		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, err = reader.Read(dat)
		if err != nil {
			return fmt.Errorf("container logs,Read data error: %w", err)
		}
		_, err = io.Copy(&buffer, bytes.NewReader(dat))
		if err != nil {
			return fmt.Errorf("container logs,Copy data error: %w", err)
		}
		if filter != nil {
			if filter(string(dat)) {
				return nil
			}
		}
	}
}

// ContainerImageSha256
//
//	@Description: 获取容器镜像sha256
//	@receiver d
//	@param ctx
//	@return []types.Container
//	@return error
func (d *Docker) ContainerImageSha256(ctx context.Context, name string) (string, error) {
	inspect, err := d.Inspect(ctx, name)
	if err != nil {
		return "", fmt.Errorf("inspect container, error: %w", err)
	}
	return inspect.Image, nil
}

// ParserYamlFiles
//
//	@Description: 解析yaml文件
//	@receiver d
//	@param ctx
//	@param yamlFiles
//	@return map[string][]string
//	@return error
func (d *Docker) ParserYamlFiles(ctx context.Context, yamlFiles ...string) (map[string][]string, error) {
	if len(yamlFiles) == 0 {
		return nil, ErrNoYaml
	}
	list, err := d.ComposeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取compose stack列表失败, err: %w", err)
	}

	sort.Strings(yamlFiles)

	projects := make(map[string]string)
	for _, stack := range list {
		projects[stack.Name] = stack.ConfigFiles
	}

	sortProjects := make(map[string][]string)
	for pg, s := range projects {
		configs := strings.Split(s, ",")
		sort.Strings(configs)
		sortProjects[pg] = configs
	}
	return sortProjects, nil
}

// GetDockerBridge 获取docker0的IP地址
func (d *Docker) GetDockerBridge(ctx context.Context) (string, error) {
	n, err := d.NetworkListByName(ctx, "bridge")
	if err != nil {
		return "", fmt.Errorf("Docker.NetworkListByName(),err:%w", err)
	}
	if len(n.IPAM.Config) == 0 {
		return "", ErrSubnetGateway
	}
	return n.IPAM.Config[0].Gateway, nil
}

func (d *Docker) svcInComposeFile(ctx context.Context, file, service string) (*types2.ServiceConfig, bool, error) {
	projectObj, err := ComposeYamlRead(ctx, file, d.EnvFiles...)
	if err != nil {
		return nil, false, err
	}

	for _, s := range projectObj.Services {
		if s.Name == service {
			return &s, true, nil
		}
	}

	return nil, false, nil
}

// matchAuthConfig
//
//	@Description: 通过镜像名匹配账户信息
//	@receiver d
//	@param regImage
//	@return string
//	@return error
func (d *Docker) matchAuthConfig(regImage string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(regImage)
	if err != nil {
		return "", fmt.Errorf("parse镜像:%s error: %w", regImage, err)
	}

	authConfigs, err := encodedAuth(ref, d.DockerCLIClient.ConfigFile())
	if err != nil {
		return "", fmt.Errorf("encode auth, error: %w", err)
	}
	return authConfigs, nil
}

// imageLoadFromIOReader
//
//	@Description: 从IO中读取镜像
//	@receiver d
//	@param input
//	@return error
func (d *Docker) imageLoadFromIOReader(ctx context.Context, input io.Reader) error {
	_, err := d.DockerCLIClient.Client().ImageLoad(ctx, input, client.ImageLoadWithQuiet(true))
	if err != nil {
		return fmt.Errorf("加载镜像 error: %w", err)
	}
	return nil
}
