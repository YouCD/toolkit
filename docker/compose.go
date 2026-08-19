package docker

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	types2 "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/youcd/toolkit/log"
)

// ComposeLoadProjectFromYaml
//
//	@Description:从 yaml中加载 Project
//	@receiver d
//	@param yamlFiles
//	@return *types2.Project
//	@return error
func (d *Docker) ComposeLoadProjectFromYaml(ctx context.Context, projectName string, normalization bool, otherLabel map[string]string, yamlFiles ...string) (*types2.Project, error) {
	opts, err := cli.NewProjectOptions(
		yamlFiles,
		cli.WithName(projectName),
		// 防止创建默认网络
		cli.WithNormalization(normalization),
		cli.WithEnvFiles(d.EnvFiles...),
		cli.WithDotEnv,
		cli.WithLoadOptions(loader.WithSkipValidation),
	)
	if err != nil {
		return nil, fmt.Errorf("new project options, error: %w", err)
	}
	p, err := opts.LoadProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("project from options, error: %w", err)
	}
	d.defaultLabel(projectName, p, otherLabel, yamlFiles...)
	return p, nil
}

// ComposeStopAllProject
//
//	@Description: compose stop
//	@receiver d
//	@param ctx
//	@param yamlFiles
//	@return error
func (d *Docker) ComposeStopAllProject(ctx context.Context, otherLabel map[string]string, yamlFiles ...string) error {
	sortProjects, err := d.ParserYamlFiles(ctx, yamlFiles...)
	if err != nil {
		return fmt.Errorf("ParserYamlFiles() error: %w", err)
	}
	for projectName, configs := range sortProjects {
		if slice.Equal(configs, yamlFiles) {
			project, err := d.ComposeLoadProjectFromYaml(ctx, projectName, true, otherLabel, yamlFiles...)
			if err != nil {
				return err
			}

			if err := d.ComposeService.Stop(ctx, projectName, api.StopOptions{Project: project}); err != nil {
				return fmt.Errorf("ComposeService.Stop(),project:%s yamls:%s,err: %w", projectName, strings.Join(yamlFiles, ","), err)
			}
			log.WithCtx(ctx).Infof("ComposeService.Stop(),project:%s yaml:%s", projectName, strings.Join(yamlFiles, ","))
			return nil
		}
	}
	return nil
}

// defaultLabel
//
//	@Description: 添加标签
//	@receiver d
//	@param projectName
//	@param p
//	@param yamlFiles
func (d *Docker) defaultLabel(projectName string, p *types2.Project, otherLabel map[string]string, yamlFiles ...string) {
	for index, serviceObj := range p.Services {
		label := map[string]string{
			api.ProjectLabel:     projectName,
			api.VersionLabel:     "2.23.3",
			api.OneoffLabel:      "False",
			api.ServiceLabel:     serviceObj.Name,
			api.WorkingDirLabel:  path.Dir(yamlFiles[0]),
			api.ConfigFilesLabel: strings.Join(yamlFiles, ","),
		}
		if otherLabel != nil {
			for k, v := range otherLabel {
				label[k] = v
			}
		}
		serviceConfig := p.Services[index]
		serviceConfig.CustomLabels = label
		p.Services[index] = serviceConfig
	}
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
		// Timeout:  nil,
		// Services: nil,
		// NoDeps:   false,
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

// ComposeDownAllProject
//
//	@Description: compose down
//	@receiver d
//	@param yamlFiles
//	@return error
func (d *Docker) ComposeDownAllProject(ctx context.Context, yamlFiles ...string) error {
	sortProjects, err := d.ParserYamlFiles(ctx, yamlFiles...)
	if err != nil {
		return fmt.Errorf("ParserYamlFiles() error: %w", err)
	}

	for projectName := range sortProjects {
		project, err := d.ComposeLoadProjectFromYaml(ctx, projectName, true, nil, yamlFiles...)
		if err != nil {
			return err
		}
		if err := d.ComposeService.Down(ctx, projectName, api.DownOptions{Project: project}); err != nil {
			return fmt.Errorf("ComposeService.Down(),project:%s yamls:%s,err: %w", projectName, strings.Join(yamlFiles, ","), err)
		}
		return nil
	}

	return nil
}

// ComposeUp
//
//	@Description: 启动
//	@receiver d
//	@param yamlFiles
//	@return error
func (d *Docker) ComposeUp(ctx context.Context, projectName string, recreateMod string, normalization bool, otherLabel map[string]string, yamlFiles ...string) error {
	p, err := d.ComposeLoadProjectFromYaml(ctx, projectName, normalization, otherLabel, yamlFiles...)
	if err != nil {
		return fmt.Errorf("load yaml, error: %w", err)
	}
	//  设置镜像拉取策略
	for i, service := range p.Services {
		service.PullPolicy = "never"
		p.Services[i] = service
	}
	err = d.ComposeServiceUp(ctx, p, recreateMod)
	if err != nil {
		return fmt.Errorf("compose up error: %w", err)
	}
	return nil
}
