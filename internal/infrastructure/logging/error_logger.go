package logging

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
	
	"github.com/hashicorp/go-hclog"
	
	"github.com/cloudflare/tf-migrate/internal/core"
)

// ErrorLogger provides structured error logging
type ErrorLogger struct {
	logger hclog.Logger
	config *ErrorLogConfig
}

// ErrorLogConfig configures error logging behavior
type ErrorLogConfig struct {
	IncludeStackTrace   bool
	IncludeContext      bool
	IncludeTimestamp    bool
	IncludeErrorType    bool
	MaxContextDepth     int
	SensitiveKeys       []string // Keys to redact from context
	GroupSimilarErrors  bool
	ErrorCountThreshold int // Log warning when this many errors occur
}

// DefaultErrorLogConfig returns sensible defaults
func DefaultErrorLogConfig() *ErrorLogConfig {
	return &ErrorLogConfig{
		IncludeStackTrace:   true,
		IncludeContext:      true,
		IncludeTimestamp:    true,
		IncludeErrorType:    true,
		MaxContextDepth:     3,
		SensitiveKeys:       []string{"password", "token", "secret", "key", "credential"},
		GroupSimilarErrors:  true,
		ErrorCountThreshold: 10,
	}
}

// NewErrorLogger creates a new error logger
func NewErrorLogger(logger hclog.Logger, config *ErrorLogConfig) *ErrorLogger {
	if config == nil {
		config = DefaultErrorLogConfig()
	}
	return &ErrorLogger{
		logger: logger,
		config: config,
	}
}

// LogError logs an error with full context
func (el *ErrorLogger) LogError(err error) {
	if err == nil {
		return
	}
	
	// Check if it's a MigrationError for rich context
	if migErr, ok := err.(*core.MigrationError); ok {
		el.logMigrationError(migErr)
		return
	}
	
	// Check if it's an ErrorList
	if errList, ok := err.(*core.ErrorList); ok {
		el.logErrorList(errList)
		return
	}
	
	// Log as generic error
	el.logger.Error("Error occurred", "error", err.Error())
}

// logMigrationError logs a MigrationError with all its context
func (el *ErrorLogger) logMigrationError(err *core.MigrationError) {
	fields := []interface{}{
		"error", err.Error(),
	}
	
	if el.config.IncludeErrorType {
		fields = append(fields, "error_type", err.Type.String())
	}
	
	if el.config.IncludeTimestamp {
		fields = append(fields, "timestamp", time.Now().Format(time.RFC3339))
	}
	
	if err.Operation != "" {
		fields = append(fields, "operation", err.Operation)
	}
	
	if err.Resource != "" {
		fields = append(fields, "resource", err.Resource)
	}
	
	if err.File != "" {
		fields = append(fields, "file", err.File)
		if err.Line > 0 {
			fields = append(fields, "line", err.Line)
		}
	}
	
	if err.Version != "" {
		fields = append(fields, "version", err.Version)
	}
	
	if err.Recoverable {
		fields = append(fields, "recoverable", true)
	}
	
	// Add context if configured
	if el.config.IncludeContext && len(err.Context) > 0 {
		sanitizedContext := el.sanitizeContext(err.Context)
		if contextJSON, err := json.Marshal(sanitizedContext); err == nil {
			fields = append(fields, "context", string(contextJSON))
		}
	}
	
	// Add stack trace if configured
	if el.config.IncludeStackTrace {
		fields = append(fields, "stack_trace", el.getStackTrace(2))
	}
	
	// Log with appropriate level
	if err.Recoverable {
		el.logger.Warn("Recoverable error occurred", fields...)
	} else {
		el.logger.Error("Error occurred", fields...)
	}
}

// logErrorList logs an ErrorList
func (el *ErrorLogger) logErrorList(errList *core.ErrorList) {
	errors := errList.GetErrors()
	
	if len(errors) == 0 {
		return
	}
	
	// Log summary
	el.logger.Error("Multiple errors occurred",
		"error_count", len(errors),
		"timestamp", time.Now().Format(time.RFC3339),
	)
	
	// Group similar errors if configured
	if el.config.GroupSimilarErrors {
		el.logGroupedErrors(errors)
	} else {
		// Log each error individually
		for i, err := range errors {
			if i >= 10 { // Limit individual logging
				el.logger.Error("Additional errors omitted",
					"remaining_count", len(errors)-10,
				)
				break
			}
			el.LogError(err)
		}
	}
	
	// Warn if threshold exceeded
	if len(errors) >= el.config.ErrorCountThreshold {
		el.logger.Warn("Error threshold exceeded",
			"threshold", el.config.ErrorCountThreshold,
			"actual_count", len(errors),
		)
	}
}

// logGroupedErrors groups and logs similar errors
func (el *ErrorLogger) logGroupedErrors(errors []error) {
	// Group errors by type
	groups := make(map[string][]error)
	
	for _, err := range errors {
		key := el.getErrorKey(err)
		groups[key] = append(groups[key], err)
	}
	
	// Log each group
	for key, groupErrors := range groups {
		if len(groupErrors) == 1 {
			el.LogError(groupErrors[0])
		} else {
			el.logger.Error("Error group",
				"error_type", key,
				"count", len(groupErrors),
				"first_error", groupErrors[0].Error(),
			)
		}
	}
}

// getErrorKey returns a key for grouping similar errors
func (el *ErrorLogger) getErrorKey(err error) string {
	if migErr, ok := err.(*core.MigrationError); ok {
		return fmt.Sprintf("%s:%s:%s", migErr.Type, migErr.Operation, migErr.Resource)
	}
	
	// Use error message prefix for grouping
	msg := err.Error()
	if idx := strings.Index(msg, ":"); idx > 0 {
		return msg[:idx]
	}
	
	return "generic_error"
}

// sanitizeContext removes sensitive information from context
func (el *ErrorLogger) sanitizeContext(context map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	
	for key, value := range context {
		// Check if key contains sensitive words
		isSensitive := false
		lowerKey := strings.ToLower(key)
		for _, sensitiveKey := range el.config.SensitiveKeys {
			if strings.Contains(lowerKey, strings.ToLower(sensitiveKey)) {
				isSensitive = true
				break
			}
		}
		
		if isSensitive {
			sanitized[key] = "[REDACTED]"
		} else {
			// Recursively sanitize nested maps
			if nestedMap, ok := value.(map[string]interface{}); ok {
				sanitized[key] = el.sanitizeContext(nestedMap)
			} else {
				sanitized[key] = value
			}
		}
	}
	
	return sanitized
}

// getStackTrace returns the current stack trace
func (el *ErrorLogger) getStackTrace(skip int) string {
	const maxFrames = 10
	
	var traces []string
	for i := skip; i < skip+maxFrames; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		// Skip runtime and standard library frames
		if strings.Contains(file, "runtime/") {
			continue
		}
		
		traces = append(traces, fmt.Sprintf("  %s:%d %s", file, line, fn.Name()))
	}
	
	return strings.Join(traces, "\n")
}

// ErrorStats tracks error statistics
type ErrorStats struct {
	TotalErrors      int                    `json:"total_errors"`
	ErrorsByType     map[string]int         `json:"errors_by_type"`
	ErrorsByFile     map[string]int         `json:"errors_by_file"`
	ErrorsByResource map[string]int         `json:"errors_by_resource"`
	RecoverableCount int                    `json:"recoverable_count"`
	StartTime        time.Time              `json:"start_time"`
	LastError        time.Time              `json:"last_error"`
}

// ErrorStatsCollector collects error statistics
type ErrorStatsCollector struct {
	stats  *ErrorStats
	logger *ErrorLogger
}

// NewErrorStatsCollector creates a new stats collector
func NewErrorStatsCollector(logger *ErrorLogger) *ErrorStatsCollector {
	return &ErrorStatsCollector{
		stats: &ErrorStats{
			ErrorsByType:     make(map[string]int),
			ErrorsByFile:     make(map[string]int),
			ErrorsByResource: make(map[string]int),
			StartTime:        time.Now(),
		},
		logger: logger,
	}
}

// RecordError records an error in statistics
func (c *ErrorStatsCollector) RecordError(err error) {
	if err == nil {
		return
	}
	
	c.stats.TotalErrors++
	c.stats.LastError = time.Now()
	
	// Track MigrationError specifics
	if migErr, ok := err.(*core.MigrationError); ok {
		c.stats.ErrorsByType[migErr.Type.String()]++
		
		if migErr.File != "" {
			c.stats.ErrorsByFile[migErr.File]++
		}
		
		if migErr.Resource != "" {
			c.stats.ErrorsByResource[migErr.Resource]++
		}
		
		if migErr.Recoverable {
			c.stats.RecoverableCount++
		}
	}
	
	// Track ErrorList
	if errList, ok := err.(*core.ErrorList); ok {
		for _, e := range errList.GetErrors() {
			c.RecordError(e)
		}
	}
}

// GetStats returns current statistics
func (c *ErrorStatsCollector) GetStats() *ErrorStats {
	return c.stats
}

// LogStats logs the current statistics
func (c *ErrorStatsCollector) LogStats() {
	if c.stats.TotalErrors == 0 {
		c.logger.logger.Info("No errors recorded")
		return
	}
	
	duration := time.Since(c.stats.StartTime)
	
	c.logger.logger.Info("Error statistics",
		"total_errors", c.stats.TotalErrors,
		"recoverable_errors", c.stats.RecoverableCount,
		"duration", duration.String(),
		"errors_per_minute", float64(c.stats.TotalErrors)/duration.Minutes(),
	)
	
	// Log top error types
	for errType, count := range c.stats.ErrorsByType {
		if count > 1 {
			c.logger.logger.Info("Errors by type",
				"type", errType,
				"count", count,
			)
		}
	}
	
	// Log problematic files
	for file, count := range c.stats.ErrorsByFile {
		if count > 2 {
			c.logger.logger.Warn("File with multiple errors",
				"file", file,
				"error_count", count,
			)
		}
	}
}

// Reset resets the statistics
func (c *ErrorStatsCollector) Reset() {
	c.stats = &ErrorStats{
		ErrorsByType:     make(map[string]int),
		ErrorsByFile:     make(map[string]int),
		ErrorsByResource: make(map[string]int),
		StartTime:        time.Now(),
	}
}