package log

import (
	"testing"

	"github.com/natefinch/lumberjack"
)

func TestSetLogLevel(t *testing.T) {
	cfg := &Config{
		//Stdout: true,
		LumberjackCfg: &lumberjack.Logger{
			Filename:   "/tmp/log/log.log",
			MaxSize:    1,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
			LocalTime:  true,
		},
	}
	Init(cfg)
	for {
		Info("hello world")
	}
}
