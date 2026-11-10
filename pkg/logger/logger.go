// pkg/logger/logger.go
package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init(level slog.Level) {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // вот он, трассировка вызова
	})
	Log = slog.New(h)

	// чтобы package-level вызовы (slog.Info из любого места проекта)
	// тоже шли через этот логгер
	slog.SetDefault(Log)
}
