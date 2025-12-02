package log

import (
	"bytes"
	"context"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	LumberjackCfg *lumberjack.Logger // 写到文件
	Stdout        bool               // 打印到控制台
}

var (
	logTmFmt    = "2006-01-02 15:04:05"
	logger      *zap.SugaredLogger
	atomicLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	logLevel    = zap.InfoLevel
	initLevel   = "debug"

	defaultConfig = &Config{
		Stdout: true,
	}
	defaultLogger = zap.New(newCore(defaultConfig), zap.AddCaller(), zap.Development()).Sugar()
)
var (
	lumberjackLogger *lumberjack.Logger
)

func LoggerIsNil() bool {
	return logger == nil
}

// SetLogLevel
//
//	@Description:默认级别是debug，实时修改日志级别
//	@param level
func SetLogLevel(level string) {
	initLevel = level
	setLogLevel()
}

func setLogLevel() {
	switch strings.ToLower(initLevel) {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "warn":
		logLevel = zap.WarnLevel
	case "error":
		logLevel = zap.ErrorLevel
	case "panic":
		logLevel = zap.PanicLevel
	case "fatal":
		logLevel = zap.FatalLevel
	default:
		logLevel = zap.InfoLevel
	}
	atomicLevel.SetLevel(logLevel)
}

func Init(cfg *Config) {
	if cfg == nil {
		cfg = defaultConfig
	}
	core := newCore(cfg)
	l := zap.New(core, zap.AddCaller(), zap.Development())
	logger = l.Sugar()
	setLogLevel()
}
func InitBuffer(logBuffer *bytes.Buffer) {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(logBuffer), zapcore.AddSync(os.Stdout)), // 打印到控制台和文件
		atomicLevel, // 日志级别
	)
	l := zap.New(core, zap.AddCaller(), zap.Development())
	logger = l.Sugar()
	setLogLevel()
}

func newCore(cfg *Config) zapcore.Core {
	// 设置级别
	atomicLevel.SetLevel(logLevel)

	var wsList []zapcore.WriteSyncer
	if cfg.Stdout {
		wsList = append(wsList, zapcore.AddSync(os.Stdout))
	}
	if cfg.LumberjackCfg != nil {
		lumberjackLogger = &lumberjack.Logger{
			Filename:   cfg.LumberjackCfg.Filename,   // 日志文件存放目录，如果文件夹不存在会自动创建
			MaxSize:    cfg.LumberjackCfg.MaxSize,    // 文件大小限制,单位100MB
			MaxBackups: cfg.LumberjackCfg.MaxBackups, // 最大保留日志文件数量
			MaxAge:     cfg.LumberjackCfg.MaxAge,     // 日志文件保留天数
			Compress:   cfg.LumberjackCfg.Compress,   // 是否压缩处理
			LocalTime:  cfg.LumberjackCfg.LocalTime,
		}
		infoFileWriteSyncer := zapcore.AddSync(lumberjackLogger)
		wsList = append(wsList, zapcore.AddSync(infoFileWriteSyncer))
	}
	return zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		//		zapcore.NewJSONEncoder(encoderConfig), // 编码器配置
		zapcore.NewMultiWriteSyncer(wsList...), // 打印到控制台和文件
		atomicLevel,                            // 日志级别
	)
}

func newEncoderConfig() zapcore.EncoderConfig {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "ts",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		EncodeLevel:   zapcore.CapitalColorLevelEncoder, // 这里可以指定颜色
		LineEnding:    zapcore.DefaultLineEnding,
		// EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写编码器
		EncodeTime:     zapcore.TimeEncoderOfLayout(logTmFmt), // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,        //
		EncodeCaller:   zapcore.ShortCallerEncoder,            // 短路径编码器
		// EncodeCaller:   zapcore.FullCallerEncoder,    // 全路径编码器
		EncodeName: zapcore.FullNameEncoder,
	}
	return encoderConfig
}

func GetLogFile() string {
	if lumberjackLogger != nil {
		return lumberjackLogger.Filename
	}
	return "no file"
}

func GetLogger() *zap.SugaredLogger {
	return logger
}

// WithCtx 返回一个带有 request_id 的 SugaredLogger
// 注意: 返回的 logger 会继承全局 logger 的 callerSkip 设置
// 在热路径上建议缓存结果: l := log.WithCtx(ctx); l.Info(...)
func WithCtx(ctx context.Context) *zap.SugaredLogger {
	var requestId string

	if id, ok := ctx.Value("request_id").(string); ok && id != "" {
		requestId = id
	}
	if requestId != "" {
		return logger.With("request_id", requestId)
	}
	if logger == nil {
		return defaultLogger
	}
	return logger
}

func SetRequestId(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, "request_id", uuid.New().String())
	return ctx
}
