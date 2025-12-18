package log

import (
	"context"
	"testing"
)

func TestSetLogLevel(t *testing.T) {
	cfg := &Config{
		Stdout: true,
		//LumberjackCfg: &lumberjack.Logger{
		//	Filename:   "/tmp/log/log.log",
		//	MaxSize:    1,
		//	MaxBackups: 5,
		//	MaxAge:     30,
		//	Compress:   true,
		//	LocalTime:  true,
		//},
	}
	Init(nil)
	WithCtx(context.Background()).Info("Info")
	SetLogLevel("debug")
	WithCtx(context.Background()).Debug("Debug")
	SetLogLevel("info")
	WithCtx(context.Background()).Debug("Debug")
	WithCtx(context.Background()).Info("Info")
}
