package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

var logger *slog.Logger

func Debug(msg string, args ...any) {
	if logger != nil {
		logger.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if logger != nil {
		logger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if logger != nil {
		logger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if logger != nil {
		logger.Error(msg, args...)
	}
}

func Init(level slog.Level) {
	w := os.Stderr

	handler := tint.NewHandler(w, &tint.Options{
		TimeFormat: time.TimeOnly,
		Level:      level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if lv, ok := a.Value.Any().(slog.Level); ok && lv == slog.LevelDebug {
					a.Value = slog.StringValue("\x1b[34mdebug\x1b[0m")
				}
				if lv, ok := a.Value.Any().(slog.Level); ok && lv == slog.LevelInfo {
					a.Value = slog.StringValue("\x1b[32minfo\x1b[0m")
				}
				if lv, ok := a.Value.Any().(slog.Level); ok && lv == slog.LevelError {
					a.Value = slog.StringValue("\x1b[31merror\x1b[0m")
				}
			}
			return a
		},
	})

	logger = slog.New(handler)
	slog.SetDefault(logger)
}
