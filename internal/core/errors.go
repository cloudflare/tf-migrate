package core

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents the type of error that occurred
type ErrorType int

const (
	// ParseError indicates an error parsing HCL or JSON
	ParseError ErrorType = iota
	// TransformError indicates an error during transformation
	TransformError
	// ValidationError indicates validation failed
	ValidationError
	// StateError indicates an error with state file operations
	StateError
	// FileError indicates file I/O errors
	FileError
	// ConfigError indicates configuration errors
	ConfigError
	// ResourceNotFoundError indicates a resource type is not supported
	ResourceNotFoundError
	// VersionError indicates version-related errors
	VersionError
	// BackupError indicates backup/restore errors
	BackupError
)

func (t ErrorType) String() string {
	switch t {
	case ParseError:
		return "Parse"
	case TransformError:
		return "Transform"
	case ValidationError:
		return "Validation"
	case StateError:
		return "State"
	case FileError:
		return "File"
	case ConfigError:
		return "Config"
	case ResourceNotFoundError:
		return "ResourceNotFound"
	case VersionError:
		return "Version"
	case BackupError:
		return "Backup"
	default:
		return "Unknown"
	}
}

// MigrationError provides rich error context for migration operations
type MigrationError struct {
	Type        ErrorType
	Operation   string                 // What operation was being performed
	Resource    string                 // Resource type being processed
	File        string                 // File being processed
	Line        int                    // Line number if applicable
	Version     string                 // Version being migrated
	Cause       error                  // Underlying error
	Context     map[string]interface{} // Additional context
	Recoverable bool                   // Whether the error is recoverable
}

// Error implements the error interface
func (e *MigrationError) Error() string {
	var parts []string
	
	// Build error message with context
	parts = append(parts, fmt.Sprintf("[%s Error]", e.Type))
	
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("during %s", e.Operation))
	}
	
	if e.Resource != "" {
		parts = append(parts, fmt.Sprintf("for resource '%s'", e.Resource))
	}
	
	if e.File != "" {
		if e.Line > 0 {
			parts = append(parts, fmt.Sprintf("in %s:%d", e.File, e.Line))
		} else {
			parts = append(parts, fmt.Sprintf("in %s", e.File))
		}
	}
	
	if e.Version != "" {
		parts = append(parts, fmt.Sprintf("(version %s)", e.Version))
	}
	
	message := strings.Join(parts, " ")
	
	if e.Cause != nil {
		message = fmt.Sprintf("%s: %v", message, e.Cause)
	}
	
	return message
}

// Unwrap returns the underlying error
func (e *MigrationError) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is
func (e *MigrationError) Is(target error) bool {
	t, ok := target.(*MigrationError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// WithContext adds context to the error
func (e *MigrationError) WithContext(key string, value interface{}) *MigrationError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// ErrorBuilder provides a fluent interface for building errors
type ErrorBuilder struct {
	err *MigrationError
}

// NewError creates a new error builder
func NewError(errType ErrorType) *ErrorBuilder {
	return &ErrorBuilder{
		err: &MigrationError{
			Type:        errType,
			Context:     make(map[string]interface{}),
			Recoverable: false,
		},
	}
}

// WithOperation sets the operation that was being performed
func (b *ErrorBuilder) WithOperation(op string) *ErrorBuilder {
	b.err.Operation = op
	return b
}

// WithResource sets the resource type
func (b *ErrorBuilder) WithResource(resource string) *ErrorBuilder {
	b.err.Resource = resource
	return b
}

// WithFile sets the file being processed
func (b *ErrorBuilder) WithFile(file string) *ErrorBuilder {
	b.err.File = file
	return b
}

// WithLine sets the line number
func (b *ErrorBuilder) WithLine(line int) *ErrorBuilder {
	b.err.Line = line
	return b
}

// WithVersion sets the version
func (b *ErrorBuilder) WithVersion(version string) *ErrorBuilder {
	b.err.Version = version
	return b
}

// WithCause sets the underlying error
func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	b.err.Cause = cause
	return b
}

// WithContext adds context information
func (b *ErrorBuilder) WithContext(key string, value interface{}) *ErrorBuilder {
	b.err.Context[key] = value
	return b
}

// Recoverable marks the error as recoverable
func (b *ErrorBuilder) Recoverable() *ErrorBuilder {
	b.err.Recoverable = true
	return b
}

// Build returns the constructed error
func (b *ErrorBuilder) Build() error {
	return b.err
}

// ErrorList accumulates multiple errors
type ErrorList struct {
	Errors []error
	limit  int
}

// NewErrorList creates a new error list with optional limit
func NewErrorList(limit int) *ErrorList {
	if limit <= 0 {
		limit = 100 // Default limit
	}
	return &ErrorList{
		Errors: make([]error, 0),
		limit:  limit,
	}
}

// Add adds an error to the list
func (l *ErrorList) Add(err error) {
	if err != nil && len(l.Errors) < l.limit {
		l.Errors = append(l.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (l *ErrorList) HasErrors() bool {
	return len(l.Errors) > 0
}

// Error implements the error interface
func (l *ErrorList) Error() string {
	if len(l.Errors) == 0 {
		return "no errors"
	}
	
	var messages []string
	for i, err := range l.Errors {
		if i >= 10 { // Show first 10 errors
			messages = append(messages, fmt.Sprintf("... and %d more errors", len(l.Errors)-10))
			break
		}
		messages = append(messages, fmt.Sprintf("  %d. %v", i+1, err))
	}
	
	return fmt.Sprintf("%d errors occurred:\n%s", len(l.Errors), strings.Join(messages, "\n"))
}

// GetErrors returns all accumulated errors
func (l *ErrorList) GetErrors() []error {
	return l.Errors
}

// First returns the first error or nil
func (l *ErrorList) First() error {
	if len(l.Errors) > 0 {
		return l.Errors[0]
	}
	return nil
}

// Common error constructors

// WrapError wraps an error with context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// ParseErrorf creates a parse error
func ParseErrorf(file string, line int, format string, args ...interface{}) error {
	return NewError(ParseError).
		WithFile(file).
		WithLine(line).
		WithOperation("parsing").
		WithCause(fmt.Errorf(format, args...)).
		Build()
}

// TransformErrorf creates a transformation error
func TransformErrorf(resource, file string, cause error, format string, args ...interface{}) error {
	return NewError(TransformError).
		WithResource(resource).
		WithFile(file).
		WithOperation("transforming").
		WithCause(cause).
		WithContext("message", fmt.Sprintf(format, args...)).
		Build()
}

// ValidationErrorf creates a validation error
func ValidationErrorf(format string, args ...interface{}) error {
	return NewError(ValidationError).
		WithOperation("validating").
		WithCause(fmt.Errorf(format, args...)).
		Build()
}

// FileErrorf creates a file error
func FileErrorf(file string, op string, err error) error {
	return NewError(FileError).
		WithFile(file).
		WithOperation(op).
		WithCause(err).
		Build()
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errType ErrorType) bool {
	var migErr *MigrationError
	if errors.As(err, &migErr) {
		return migErr.Type == errType
	}
	return false
}

// GetErrorContext extracts context from an error if it's a MigrationError
func GetErrorContext(err error) map[string]interface{} {
	var migErr *MigrationError
	if errors.As(err, &migErr) {
		return migErr.Context
	}
	return nil
}