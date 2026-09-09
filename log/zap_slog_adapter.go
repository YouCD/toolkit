package log

import (
	"context"
	"log/slog"

	"go.uber.org/zap"
)

// ZapSlogAdapter 适配 zap.SugaredLogger 到 slog.Handler
type ZapSlogAdapter struct {
	Logger *zap.SugaredLogger
}

func (a *ZapSlogAdapter) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (a *ZapSlogAdapter) Handle(_ context.Context, r slog.Record) error {
	msg := r.Message
	// 添加 attrs
	r.Attrs(func(attr slog.Attr) bool {
		msg += " " + attr.Key + "=" + attr.Value.String()
		return true
	})

	switch r.Level {
	case slog.LevelError:
		a.Logger.Error(msg)
	case slog.LevelWarn:
		a.Logger.Warn(msg)
	case slog.LevelInfo:
		a.Logger.Info(msg)
	default:
		a.Logger.Debug(msg)
	}
	return nil
}

func (a *ZapSlogAdapter) WithAttrs(_ []slog.Attr) slog.Handler {
	return a
}

func (a *ZapSlogAdapter) WithGroup(_ string) slog.Handler {
	return a
}
