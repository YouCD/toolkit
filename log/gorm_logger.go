package log

import (
	"context"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ormogger "gorm.io/gorm/logger"
)

// ZapGormLogger 自定义 GORM 日志结构体
type ZapGormLogger struct {
	SlowThreshold time.Duration
	LogLevel      ormogger.LogLevel
}

func NewGormLogger(slowThreshold time.Duration, logLevel string) *ZapGormLogger {
	SetLogLevel(logLevel)
	var l ormogger.LogLevel
	level := atomicLevel.Level()
	switch level {
	case zapcore.DebugLevel:
		l = ormogger.Info
	case zapcore.WarnLevel:
		l = ormogger.Warn
	case zapcore.ErrorLevel:
		l = ormogger.Error
	default:
		l = ormogger.Silent
	}

	return &ZapGormLogger{
		SlowThreshold: slowThreshold,
		LogLevel:      l,
	}
}

// LogMode 实现 LogMode
func (l *ZapGormLogger) LogMode(level ormogger.LogLevel) ormogger.Interface {
	switch level {
	case ormogger.Silent:
		// 设置为 Fatal 或更高，确保生产环境不输出任何常规日志
		SetLogLevel("fatal")
		l.LogLevel = ormogger.Silent
	case ormogger.Error:
		SetLogLevel("error")
		l.LogLevel = ormogger.Error
	case ormogger.Warn:
		SetLogLevel("warn")
		l.LogLevel = ormogger.Warn
	case ormogger.Info:
		// GORM 的 Info 对应查看所有 SQL，因此 Zap 必须开到 Debug
		SetLogLevel("debug")
		l.LogLevel = ormogger.Info
	default:
		SetLogLevel("info")
		l.LogLevel = ormogger.Silent
	}
	return l
}

// Info 实现 Info
func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	WithCtx(ctx).Infof(msg, data...)
}

// Warn 实现 Warn
func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	WithCtx(ctx).Infof(msg, data...)
}

// Error 实现 Error
func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	WithCtx(ctx).Errorf(msg, data...)
}

func getBusinessCaller() (string, int) {
	pcs := make([]uintptr, 20)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "gorm.io") &&
			!strings.Contains(frame.File, "runtime/") &&
			!strings.Contains(frame.File, "zap") &&
			!strings.Contains(frame.File, "gorm_logger.go") {
			return frame.File, frame.Line
		}
		if !more {
			break
		}
	}
	return "unknown", 0
}

// Trace 实现 Trace，用于打印 SQL

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= ormogger.Silent {
		return
	}
	callerFile, callerLine := getBusinessCaller()
	// 构建 Entry
	entry := zapcore.Entry{
		Level:   logLevel,
		Time:    time.Now(),
		Caller:  zapcore.EntryCaller{Defined: true, File: callerFile, Line: callerLine},
		Message: "GORM",
	}
	ce := WithCtx(ctx).Desugar().Core().Check(entry, nil)
	if ce == nil {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	// 因为 WithCtx 内部已经有了一层逻辑，这里额外跳过
	switch {
	case err != nil && l.LogLevel >= ormogger.Error:
		ce.Entry.Level = zap.ErrorLevel
		ce.Write(zap.String("SQL", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed), zap.Error(err))
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= ormogger.Warn:
		ce.Write(zap.String("SQL", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed))
	case l.LogLevel >= ormogger.Info:
		ce.Write(zap.String("SQL", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed))
	}
}
