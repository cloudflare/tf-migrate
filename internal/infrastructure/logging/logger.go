package logging

import (
	"os"

	"github.com/hashicorp/go-hclog"
)

// New creates a new logger with default settings
// Always shows all output as requested
func New() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:       "tf-migrate",
		Level:      hclog.Debug, // Show all debug output
		Output:     os.Stderr,
		JSONFormat: false,
		Color:      hclog.AutoColor,
	})
}