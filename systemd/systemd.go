package systemd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

var (
	ErrServiceNotExist = errors.New("service not exist")
	ErrTimeout         = errors.New("超时")
)

type ConnectionState struct {
	Connected     bool // 是否连接
	TotalAttempts int  // 尝试连接次数
	Err           error
}
type UnitCheck struct {
	UnitName   string
	CheckCount int
}
type Msg struct {
	UnitName string
	MsgStr   string
}
type Systemd struct {
	conn                *dbus.Conn
	dbusMsgChan         chan string
	MsgChan             chan Msg
	connectionStateChan chan ConnectionState
	mux                 sync.Mutex
	unitCurrent         string
	totalConnect        int
	unitCheckChan       chan<- UnitCheck
}

var (
	conn *dbus.Conn
	once sync.Once
	s    *Systemd
)

func NewSystemd(ctx context.Context, connectionStateChan chan ConnectionState, msgChan chan Msg, unitCheckChan chan UnitCheck) (*Systemd, error) {
	var err error
	once.Do(func() {
		conn, err = dbus.NewSystemdConnectionContext(ctx)
		dbusMsgChan := make(chan string)
		s = &Systemd{conn: conn, dbusMsgChan: dbusMsgChan}
		// 异步消息
		go s.handlerMsgChan()
		// 疯狂连接
		go s.guardConnected(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("systemd connect error: %w", err)
	}
	s.connectionStateChan = connectionStateChan
	s.MsgChan = msgChan
	s.unitCheckChan = unitCheckChan
	return s, nil
}

// UnitStart
//
//	@Description: systemctl enable --now 服务；会在300s内检查服务
//	@param service
//	@return error
func (s *Systemd) UnitStart(ctx context.Context, service string) error {
	var errs []error
	s.mux.Lock()
	defer s.mux.Unlock()
	s.unitCurrent = service
	err := s.DaemonReload(ctx)
	if err != nil {
		errs = append(errs, err)
	}
	_, err = s.conn.StartUnitContext(ctx, service, "replace", s.dbusMsgChan)
	if err != nil {
		errs = append(errs, err)
	}
	// 2 状态检查
	errs = append(errs, s.unitCheck(ctx, service))
	if len(errs) > 0 {
		return errors.Join(err)
	}
	return nil
}

// EnableService
//
//	@Description: enable 服务
//	@receiver s
//	@param services
//	@return error
func (s *Systemd) EnableService(ctx context.Context, services ...string) error {
	_ = s.DaemonReload(ctx)
	_, _, err := s.conn.EnableUnitFilesContext(ctx, services, false, true)
	if err != nil {
		return fmt.Errorf("enable service %s, error:%w", services, err)
	}

	return nil
}

// UnitStartWithEnable
//
//	@Description: start enable 服务
//	@receiver s
//	@param service
//	@return error
func (s *Systemd) UnitStartWithEnable(ctx context.Context, service string) error {
	err := s.UnitStart(ctx, service)
	if err != nil {
		return fmt.Errorf("start service %s, error:%w", service, err)
	}
	// enable
	return s.EnableService(ctx, service)
}

func (s *Systemd) Close() error {
	close(s.dbusMsgChan)
	s.conn.Close()
	return nil
}

// UnitRestart
//
//	@Description: 重启服务
//	@param service
//	@return error
func (s *Systemd) UnitRestart(ctx context.Context, service string) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.unitCurrent = service

	_, err := s.conn.RestartUnitContext(ctx, service, "replace", s.dbusMsgChan)
	if err != nil {
		return fmt.Errorf("restart service %s, error:%w", service, err)
	}

	time.Sleep(time.Second * 3)

	return s.unitCheck(ctx, service)
}

// UnitFragmentPath
//
//	@Description: 获取service文件路径
//	@receiver s
//	@param service
//	@return string
//	@return error
func (s *Systemd) UnitFragmentPath(ctx context.Context, service string) (string, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.unitCurrent = service
	return s.UnitSomeProperty(ctx, service, "FragmentPath")
}

var ErrServicePropertyNotFound = errors.New("服务属性不存在")

func (s *Systemd) UnitSomeProperty(ctx context.Context, service, someProperty string) (string, error) {
	property, err := s.conn.GetUnitPropertiesContext(ctx, service)
	if err != nil {
		return "", ErrServicePropertyNotFound
	}
	value, ok := property[someProperty].(string)
	if ok {
		return value, nil
	}
	return "", ErrServicePropertyNotFound
}

// UnitIsActive
//
//	@Description: 服务是否启动
//	@param service
//	@param sshClient
//	@return bool
func (s *Systemd) UnitIsActive(ctx context.Context, service string) bool {
	someProperty, err := s.UnitSomeProperty(ctx, service, "ActiveState")
	if err != nil {
		return false
	}
	return someProperty == "active"
}

// UnitIsActiveStatus
//
//	@Description: systemctl is-active 服务
//	@param name
//	@param sshClient
//	@return status
//	@return err
func (s *Systemd) UnitIsActiveStatus(ctx context.Context, service string) ServiceStatus {
	var status ServiceStatus
	someProperty, err := s.UnitSomeProperty(ctx, service, "ActiveState")
	if err != nil {
		return ServiceStatusUnknown
	}
	word := strings.TrimSpace(strings.ToLower(someProperty))
	switch word {
	case "activating":
		status = ServiceStatusActivating
	case "active":
		status = ServiceStatusActivate
	case "unknown":
		status = ServiceStatusUnknown
	default:
		status = ServiceStatusOther
	}
	return status
}

// UnitDisableAndMask
//
//	@Description: systemctl disable --now 服务 && systemctl mask 服务
//	@param service
//	@return error
func (s *Systemd) UnitDisableAndMask(ctx context.Context, services ...string) error {
	var errs []error
	// stop
	for _, service := range services {
		err := s.UnitStop(ctx, service)
		if err != nil {
			errs = append(errs, fmt.Errorf("停止失败: %s, err: %w", service, err))
		}
	}

	_, err := s.conn.DisableUnitFilesContext(ctx, services, false)
	if err != nil {
		return fmt.Errorf("systemctl disable %s err: %w", services, err)
	}
	errs = append(errs, s.UnitMask(ctx, services...))

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *Systemd) UnitStopDisable(ctx context.Context, services ...string) error {
	var errs []error
	// stop
	for _, service := range services {
		err := s.UnitStop(ctx, service)
		if err != nil {
			errs = append(errs, fmt.Errorf("停止失败: %s, err: %w", service, err))
		}
	}

	// disable
	_, err := s.conn.DisableUnitFilesContext(ctx, services, false)
	if err != nil {
		errs = append(errs, fmt.Errorf("关闭失败: %s, err: %w", strings.Join(services, ","), err))
	}
	_, err = s.conn.DisableUnitFilesContext(ctx, services, false)
	if err != nil {
		return fmt.Errorf("关闭失败: %s, err: %w", strings.Join(services, ","), err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *Systemd) UnitMask(ctx context.Context, services ...string) error {
	_, err := s.conn.MaskUnitFilesContext(ctx, services, false, true)
	if err != nil {
		return fmt.Errorf("mask 失败: %s, err: %w", strings.Join(services, ","), err)
	}
	return nil
}

func (s *Systemd) UnitStop(ctx context.Context, service string) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.unitCurrent = service

	_, err := s.conn.StopUnitContext(ctx, service, "replace", s.dbusMsgChan)
	if err != nil {
		return fmt.Errorf("停止失败: %s, err: %w", service, err)
	}

	return nil
}

func (s *Systemd) DaemonReload(ctx context.Context) error {
	err := s.conn.ReloadContext(ctx)
	if err != nil {
		return fmt.Errorf("reload失败, err: %w", err)
	}
	return nil
}

// UnitListFilterByName
//
//	@Description: systemctl list-units| grep
//	@param service
//	@return []types.Service
//	@return error
func (s *Systemd) UnitListFilterByName(ctx context.Context, service string) (*dbus.UnitStatus, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.unitCurrent = service

	units, err := s.conn.ListUnitsContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListUnits失败, err: %w", err)
	}
	for _, unit := range units {
		if unit.Name == service {
			return &unit, nil
		}
	}
	return nil, ErrServiceNotExist
}

// guardConnected
//
//	@Description:疯狂连接
//	@receiver s
func (s *Systemd) guardConnected(ctx context.Context) {
	retryInterval := time.Second
	for {
		select {
		case <-ctx.Done():
			return // context canceled, exit the loop
		default:
			//nolint:nestif
			if s.conn == nil || !s.conn.Connected() {
				s.mux.Lock()
				s.totalConnect++
				s.mux.Unlock()

				conn, err := dbus.NewSystemdConnectionContext(ctx)
				if err != nil {
					// 连接失败，通过通道向调用者发送状态信息
					if s.connectionStateChan != nil {
						s.connectionStateChan <- ConnectionState{
							Connected:     false,
							TotalAttempts: s.totalConnect,
							Err:           err,
						}
					}
					// 指数回退策略
					time.Sleep(retryInterval)
					retryInterval *= 2
					if retryInterval > time.Minute {
						retryInterval = time.Minute // 限制最大重试间隔
					}
					continue
				}
				s.conn = conn
				if s.connectionStateChan != nil {
					s.connectionStateChan <- ConnectionState{
						Connected:     true,
						TotalAttempts: s.totalConnect,
					}
				}
				// 连接成功后重置重试间隔
				retryInterval = time.Second
			} else {
				// 连接正常时睡眠一段时间，避免占用CPU
				time.Sleep(time.Second * 10)
			}
		}
	}
}

// unitCheck
//
//	@Description: 检查服务
//	@param service
//	@return error
func (s *Systemd) unitCheck(ctx context.Context, service string) error {
	// 2 状态检查  ActiveState
	tick := time.NewTicker(time.Second * 300)
	defer tick.Stop()
	count := 0
	for {
		select {
		case <-tick.C:
			return fmt.Errorf("服务: %s  检查超时. err:%w", service, ErrTimeout)
		default:
			count++
			if s.unitCheckChan != nil {
				s.unitCheckChan <- UnitCheck{
					UnitName:   service,
					CheckCount: count,
				}
			}

			someProperty, err := s.UnitSomeProperty(ctx, service, "ActiveState")
			if err != nil {
				return fmt.Errorf("service: %s, error: %w", service, err)
			}
			if someProperty == "active" {
				return nil
			}

			// 如果没有启动
			if someProperty != "active" {
				_, _ = s.conn.StartUnitContext(ctx, service, "replace", s.dbusMsgChan)
				time.Sleep(5 * time.Second)
				continue
			}
		}
	}
}
func (s *Systemd) handlerMsgChan() {
	for msg := range s.dbusMsgChan {
		if s.MsgChan != nil {
			s.MsgChan <- Msg{
				UnitName: s.unitCurrent,
				MsgStr:   msg,
			}
		}
	}
}
