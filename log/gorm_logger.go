package log

import (
	"context"
	"fmt"
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

			// 提取短路径：只保留最后两个目录和文件名
			shortPath := extractShortPath(frame.File)
			return shortPath, frame.Line
		}
		if !more {
			break
		}
	}
	return "unknown", 0
}

// extractShortPath 提取短路径，只保留最后的部分
func extractShortPath(fullPath string) string {
	// 尝试找到项目相关的目录
	// 例如: /home/xxx/project/internal/repository/file.go -> repository/file.go
	parts := strings.Split(fullPath, "/")

	// 如果路径部分少于2个，返回原路径
	if len(parts) < 2 {
		return fullPath
	}

	// 查找关键目录如 internal, pkg, cmd 等
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "internal" || parts[i] == "pkg" || parts[i] == "cmd" || parts[i] == "api" {
			// 返回从关键目录的下一级开始的路径（跳过关键目录本身）
			if i+1 < len(parts) {
				return strings.Join(parts[i+1:], "/")
			}
		}
	}

	// 如果没有找到关键目录，返回最后2个部分
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}

	return fullPath
}

// colorizeSQLType 根据 SQL 类型返回带颜色的 SQL 语句
func colorizeSQLType(sql string) string {
	upperSQL := strings.ToUpper(strings.TrimSpace(sql))
	var colorCode string

	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		colorCode = "\033[36m" // Cyan for SELECT
	case strings.HasPrefix(upperSQL, "INSERT"):
		colorCode = "\033[32m" // Green for INSERT
	case strings.HasPrefix(upperSQL, "UPDATE"):
		colorCode = "\033[33m" // Yellow for UPDATE
	case strings.HasPrefix(upperSQL, "DELETE"):
		colorCode = "\033[31m" // Red for DELETE
	default:
		colorCode = "\033[37m" // White for others
	}

	resetCode := "\033[0m"
	return fmt.Sprintf("%s%s%s", colorCode, sql, resetCode)
}

// colorizeDuration 根据执行时间返回带颜色的持续时间
func colorizeDuration(duration time.Duration, threshold time.Duration) string {
	resetCode := "\033[0m"

	if threshold > 0 && duration > threshold {
		return fmt.Sprintf("\033[31m%v%s", duration, resetCode) // Red for slow queries
	} else if duration > 100*time.Millisecond {
		return fmt.Sprintf("\033[33m%v%s", duration, resetCode) // Yellow for moderate
	}
	return fmt.Sprintf("\033[32m%v%s", duration, resetCode) // Green for fast
}

// Trace 实现 Trace，用于打印 SQL

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= ormogger.Silent {
		return
	}
	callerFile, callerLine := getBusinessCaller()

	elapsed := time.Since(begin)
	sql, rows := fc()

	coloredSQL := colorizeSQLType(sql)
	coloredDuration := colorizeDuration(elapsed, l.SlowThreshold)

	// 获取基础 logger
	base := GetLogger()
	if base == nil {
		// 如果全局 logger 未初始化，使用默认 logger
		base = defaultLogger
	}

	// 创建一个不带 caller 的 logger，避免显示 GORM 内部堆栈
	logLogger := base.Desugar().WithOptions(zap.WithCaller(false)).Sugar()

	// 保留 request_id 上下文
	if ctx != nil {
		if requestId, ok := ctx.Value("request_id").(string); ok && requestId != "" {
			logLogger = logLogger.With("request_id", requestId)
		}
	}

	// 构建带颜色的消息
	switch {
	case err != nil && l.LogLevel >= ormogger.Error:
		msg := fmt.Sprintf("\033[35mGORM\033[0m [ERROR] %s:%d | SQL: %s | rows: %d | elapsed: %s | error: %v",
			callerFile, callerLine, coloredSQL, rows, coloredDuration, err)
		logLogger.Desugar().Error(msg)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= ormogger.Warn:
		msg := fmt.Sprintf("\033[35mGORM\033[0m [SLOW] %s:%d | SQL: %s | rows: %d | elapsed: %s",
			callerFile, callerLine, coloredSQL, rows, coloredDuration)
		logLogger.Desugar().Warn(msg)
	case l.LogLevel >= ormogger.Info:
		msg := fmt.Sprintf("\033[35mGORM\033[0m %s:%d | SQL: %s | rows: %d | elapsed: %s",
			callerFile, callerLine, coloredSQL, rows, coloredDuration)
		logLogger.Desugar().Info(msg)
	}
}
