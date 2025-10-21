package logger

import (
	"os"

	"github.com/hashicorp/go-hclog"
)

var log hclog.Logger

func Init(verbose bool, debug bool, quiet bool) {
	level := hclog.Info
	if quiet {
		level = hclog.Error
	} else if debug {
		level = hclog.Debug
	} else if verbose {
		level = hclog.Trace
	}

	log = hclog.New(&hclog.LoggerOptions{
		Name:       "tf-migrate",
		Level:      level,
		Output:     os.Stderr,
		JSONFormat: false,
		Color:      hclog.AutoColor,
	})
}

func Debug(msg string, args ...interface{}) {
	if log == nil {
		return
	}
	log.Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	if log == nil {
		return
	}
	log.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	if log == nil {
		return
	}
	log.Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	if log == nil {
		return
	}
	log.Error(msg, args...)
}
