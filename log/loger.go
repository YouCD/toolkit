package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logTmFmt    = "2006-01-02 15:04:05"
	logger      *zap.SugaredLogger
	atomicLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	logLevel    = zap.InfoLevel
	LogLevel    = "debug"

	// 打印到文件
	format   = time.Now().Format("20060102")
	fileName = fmt.Sprintf("install_%s.log", format)
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

// SetFileName
//
//	@Description: 设置文件名
//	@param file
func SetFileName(file string) {
	fileName = file
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

func Init(stdout bool) {
	core := newCore(stdout, false)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.Development())
	logger = l.Sugar()
	setLogLevel()
}
func InitBuffer(logBuffer *bytes.Buffer) {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(logBuffer), zapcore.AddSync(os.Stdout)), // 打印到控制台和文件
		atomicLevel, // 日志级别
	)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.Development())
	logger = l.Sugar()
	setLogLevel()
}

func InitLogBoth() {
	core := newCore(true, true)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.Development())
	logger = l.Sugar()
	setLogLevel()
}

func newCore(stdout bool, both bool) zapcore.Core {
	// 设置级别
	atomicLevel.SetLevel(logLevel)

	var wsList []zapcore.WriteSyncer
	switch {
	case both:
		// 打印到控制台
		wsList = append(wsList, zapcore.AddSync(os.Stdout))
		// 写到文件
		f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		wsList = append(wsList, zapcore.AddSync(f))
	case stdout && !both:
		// 打印到控制台
		wsList = append(wsList, zapcore.AddSync(os.Stdout))
	default:
		f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		wsList = append(wsList, zapcore.AddSync(f))
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
	dir, _ := os.Getwd()
	return filepath.Join(dir, fileName)
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
