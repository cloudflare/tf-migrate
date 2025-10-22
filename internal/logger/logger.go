package logger

import (
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// New creates a new logger instance with the specified level
// Valid levels: "debug", "info", "warn", "error", "off"
// Empty string defaults to "warn"
func New(level string) hclog.Logger {
	var logLevel hclog.Level

	switch strings.ToLower(level) {
	case "debug":
		logLevel = hclog.Debug
	case "info":
		logLevel = hclog.Info
	case "warn", "":
		logLevel = hclog.Warn
	case "error":
		logLevel = hclog.Error
	case "off":
		logLevel = hclog.Off
	default:
		// Default to warn for unknown levels
		logLevel = hclog.Warn
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       "tf-migrate",
		Level:      logLevel,
		Output:     os.Stderr,
		JSONFormat: false,
		Color:      hclog.AutoColor,
	})
}
