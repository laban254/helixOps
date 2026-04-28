package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init initializes the global slog logger with JSON handler and a service field.
func Init(serviceName, level string) {
	lvl := levelFromString(level)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	logger := slog.New(handler).With("service", serviceName)
	slog.SetDefault(logger)
}

func levelFromString(level string) slog.Leveler {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
