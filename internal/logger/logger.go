package logger

import (
	"log/slog"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Init(dir string) *lumberjack.Logger {
	logFilePath := filepath.Join(dir, "ytmusic-tui.log")

	rotator := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	handler := slog.NewJSONHandler(rotator, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return rotator
}
