package bar

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pterm/pterm"
)

type State int

const (
	StateInfo State = iota + 1
	StateWarning
	StateFail
	StateSuccess
)

func (b State) String() string {
	switch b {
	case StateWarning:
		return "Warning"
	case StateFail:
		return "Fail"
	case StateSuccess:
		return "Success"
	case StateInfo:
		return "Info"
	default:
		return "Info"
	}
}

type Bar struct {
	multiBar MultiPrinter
	barMap   sync.Map
	event    sync.Map
	name     string
}

func NewBar(name string) *Bar {
	b := &Bar{
		multiBar: DefaultMultiPrinter,
		barMap:   sync.Map{},
		event:    sync.Map{},
		name:     name,
	}
	b.run()
	return b
}

// AddStartBar
//
//	@Description: 添加一个 Star bar
//	@receiver b
//	@param barName
//	@param startMsg
func (b *Bar) AddStartBar(barName, startMsg string) {
	if br := b.mapLoad(barName); br != nil {
		bar, _ := br.WithWriter(b.multiBar.NewWriter()).Start(startMsg)
		b.barMap.Store(barName, bar)
		return
	}
	bar, _ := pterm.DefaultSpinner.WithWriter(b.multiBar.NewWriter()).Start(startMsg)

	b.barMap.Store(barName, bar)
}

// Info
//
//	@Description: 添加一个info bar
//	@receiver b
//	@param info
func (b *Bar) Info(info string) {
	pterm.DefaultSpinner.WithWriter(b.multiBar.NewWriter()).Info(info)
}

// Warning
//
//	@Description: 添加一个warning bar
//	@receiver b
//	@param warning
func (b *Bar) Warning(warning string) {
	pterm.DefaultSpinner.WithWriter(b.multiBar.NewWriter()).Warning(warning)
}

// Fail
//
//	@Description: 添加一个fail bar
//	@receiver b
//	@param fail
func (b *Bar) Fail(fail string) {
	pterm.DefaultSpinner.WithWriter(b.multiBar.NewWriter()).Fail(fail)
}

// Success
//
//	@Description: 添加一个 success bar
//	@receiver b
//	@param success
func (b *Bar) Success(success string) {
	pterm.DefaultSpinner.WithWriter(b.multiBar.NewWriter()).Success(success)
}

var once sync.Once

// run
//
//	@Description:运行bar
//	@receiver b
func (b *Bar) run() {
	once.Do(func() {
		b.Info(b.name)
	})
	start, _ := pterm.DefaultSpinner.WithWriter(b.multiBar.Writer).Start(fmt.Sprintf("%s...", b.name))

	_, _ = b.multiBar.Start()
	b.barMap.Store(b.name, start)
}

// StopStartBar
//
//	@Description: 停止一个 Start bar
//	@receiver b
//	@param barName
func (b *Bar) StopStartBar(barName string) {
	value, ok := b.barMap.Load(barName)
	if ok {
		//nolint:forcetypeassert
		_ = value.(*pterm.SpinnerPrinter).Stop()
		return
	}
}

// Stop
//
//	@Description: 停止 bar
//	@receiver b
//
//nolint:forcetypeassert
func (b *Bar) Stop() {
	msg := make([]string, 0)
	b.event.Range(func(key, value interface{}) bool {
		if value == StateFail {
			msg = append(msg, fmt.Sprintf("%s:%s", key.(string), value))
		}
		return true
	})
	if len(msg) == 0 {
		b.mapLoad(b.name).Success(b.name+": ", "执行完成")
	} else {
		b.mapLoad(b.name).Warning(b.name+": ", strings.Join(msg, ", "))
	}
	_, _ = b.multiBar.Stop()
}

//nolint:forcetypeassert
func (b *Bar) mapLoad(name string) *pterm.SpinnerPrinter {
	value, ok := b.barMap.Load(name)
	if !ok {
		return nil
	}
	return value.(*pterm.SpinnerPrinter)
}

// SetBarState
//
//	@Description:设置bar的状态
//	@receiver b
//	@param barName
//	@param msg
//	@param barState
func (b *Bar) SetBarState(barName, msg string, barState State) {
	if b.mapLoad(barName) != nil {
		switch barState {
		case StateInfo:
			b.mapLoad(barName).Info(msg)
			b.event.Store(msg, StateInfo)
		case StateWarning:
			b.mapLoad(barName).Warning(msg)
			b.event.Store(msg, StateWarning)
		case StateFail:
			b.mapLoad(barName).Fail(msg)
			b.event.Store(msg, StateFail)
		case StateSuccess:
			b.mapLoad(barName).Success(msg)
			b.event.Store(msg, StateSuccess)
		}
		_ = b.mapLoad(barName).Stop()
		return
	}
}

// UpdateStartBarMsg
//
//	@Description: 更新 start bar的msg
//	@receiver b
//	@param barName
//	@param msg
func (b *Bar) UpdateStartBarMsg(barName, msg string) {
	if b.mapLoad(barName) != nil {
		b.mapLoad(barName).UpdateText(msg)
		return
	}
}
