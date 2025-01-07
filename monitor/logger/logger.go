package logger

import (
	"fmt"
	"log/slog"
)

// Logs a fatal error and panics
func LogErrorFatal(module string, message string) error {
	slog.Error(module, "fatal", message)
	panic(fmt.Sprintf("%s: %s", module, message))
}
