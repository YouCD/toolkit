package log

import (
	"bytes"
	"os"
	"strings"

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
	LogLevel    = "debug"

	defaultConfig = &Config{
		Stdout: true,
	}
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
	LogLevel = level
	setLogLevel()
}

func setLogLevel() {
	switch strings.ToLower(LogLevel) {
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
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.Development())
	logger = l.Sugar()
	setLogLevel()
}
func InitBuffer(logBuffer *bytes.Buffer) {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(logBuffer), zapcore.AddSync(os.Stdout)), // 打印到控制台和文件
		atomicLevel,                                                                         // 日志级别
	)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.Development())
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
			Filename:   cfg.LumberjackCfg.Filename,   //日志文件存放目录，如果文件夹不存在会自动创建
			MaxSize:    cfg.LumberjackCfg.MaxSize,    //文件大小限制,单位100MB
			MaxBackups: cfg.LumberjackCfg.MaxBackups, //最大保留日志文件数量
			MaxAge:     cfg.LumberjackCfg.MaxAge,     //日志文件保留天数
			Compress:   cfg.LumberjackCfg.Compress,   //是否压缩处理
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

func Debug(args ...interface{}) {
	if logger != nil {
		logger.Debug(args...)
	}
}

func Debugf(template string, args ...interface{}) {
	if logger != nil {
		logger.Debugf(template, args...)
	}
}

func Debugw(msg string, keysAndValues ...interface{}) {
	if logger != nil {
		logger.Debugw(msg, keysAndValues...)
	}
}

func Info(args ...interface{}) {
	if logger != nil {
		logger.Info(args...)
	}
}

func Infof(template string, args ...interface{}) {
	if logger != nil {
		logger.Infof(template, args...)
	}
}

func Infow(msg string, keysAndValues ...interface{}) {
	if logger != nil {
		logger.Infow(msg, keysAndValues...)
	}
}

func Warn(args ...interface{}) {
	if logger != nil {
		logger.Warn(args...)
	}
}

func Warnf(template string, args ...interface{}) {
	if logger != nil {
		logger.Warnf(template, args...)
	}
}

func Warnw(msg string, keysAndValues ...interface{}) {
	if logger != nil {
		logger.Warnw(msg, keysAndValues...)
	}
}

func Error(args ...interface{}) {
	if logger != nil {
		logger.Error(args...)
	}
}

func Errorf(template string, args ...interface{}) {
	if logger != nil {
		logger.Errorf(template, args...)
	}
}

func Errorw(msg string, keysAndValues ...interface{}) {
	if logger != nil {
		logger.Errorw(msg, keysAndValues...)
	}
}

func Panic(args ...interface{}) {
	if logger != nil {
		logger.Panic(args...)
	}
}

func Panicf(template string, args ...interface{}) {
	if logger != nil {
		logger.Panicf(template, args...)
	}
}

func Panicw(msg string, keysAndValues ...interface{}) {
	if logger != nil {
		logger.Panicw(msg, keysAndValues...)
	}
}

func GetLogger() *zap.SugaredLogger {
	return logger
}
