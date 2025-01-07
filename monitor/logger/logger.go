package logger

import (
	"log/slog"
	"os"
)

// Logs a fatal error and panics
func LogErrorFatal(module string, message string) {
	slog.Error(module, "fatal", message)
	os.Exit(1)
}
