package core

import (
	"context"
	"fmt"
	"time"
)

// RecoveryStrategy defines how to handle errors
type RecoveryStrategy int

const (
	// FailFast stops on first error
	FailFast RecoveryStrategy = iota
	// ContinueOnError continues processing despite errors
	ContinueOnError
	// RetryOnError retries failed operations
	RetryOnError
)

// RecoveryConfig configures error recovery behavior
type RecoveryConfig struct {
	Strategy     RecoveryStrategy
	MaxRetries   int
	RetryDelay   time.Duration
	RetryBackoff float64 // Exponential backoff multiplier
	Timeout      time.Duration
}

// DefaultRecoveryConfig returns sensible defaults
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Strategy:     FailFast,
		MaxRetries:   3,
		RetryDelay:   time.Second,
		RetryBackoff: 2.0,
		Timeout:      5 * time.Minute,
	}
}

// RecoveryPoint represents a point where we can recover from errors
type RecoveryPoint struct {
	Name        string
	SaveFunc    func() error          // Function to save state
	RestoreFunc func() error          // Function to restore state
	CleanupFunc func()                // Function to clean up resources
	Context     map[string]interface{} // Additional context
}

// Recoverer handles error recovery
type Recoverer struct {
	config         *RecoveryConfig
	points         []RecoveryPoint
	currentPoint   int
	errorHistory   []error
	successHistory []string
}

// NewRecoverer creates a new recovery handler
func NewRecoverer(config *RecoveryConfig) *Recoverer {
	if config == nil {
		config = DefaultRecoveryConfig()
	}
	return &Recoverer{
		config:         config,
		points:         make([]RecoveryPoint, 0),
		errorHistory:   make([]error, 0),
		successHistory: make([]string, 0),
		currentPoint:   -1,
	}
}

// AddRecoveryPoint adds a new recovery point
func (r *Recoverer) AddRecoveryPoint(point RecoveryPoint) error {
	// Save the state at this point
	if point.SaveFunc != nil {
		if err := point.SaveFunc(); err != nil {
			return NewError(BackupError).
				WithOperation("saving recovery point").
				WithContext("point_name", point.Name).
				WithCause(err).
				Build()
		}
	}
	
	r.points = append(r.points, point)
	r.currentPoint = len(r.points) - 1
	r.successHistory = append(r.successHistory, point.Name)
	
	return nil
}

// Execute runs a function with recovery support
func (r *Recoverer) Execute(name string, fn func() error) error {
	return r.ExecuteWithContext(context.Background(), name, fn)
}

// ExecuteWithContext runs a function with recovery support and context
func (r *Recoverer) ExecuteWithContext(ctx context.Context, name string, fn func() error) error {
	// Create a timeout context if configured
	if r.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}
	
	var lastErr error
	attempts := 0
	maxAttempts := 1
	
	if r.config.Strategy == RetryOnError {
		maxAttempts = r.config.MaxRetries
	}
	
	delay := r.config.RetryDelay
	
	for attempts < maxAttempts {
		attempts++
		
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return NewError(ConfigError).
				WithOperation("executing with recovery").
				WithContext("operation_name", name).
				WithContext("attempts", attempts).
				WithCause(ctx.Err()).
				Build()
		default:
		}
		
		// Try to execute the function
		err := fn()
		
		if err == nil {
			// Success!
			r.successHistory = append(r.successHistory, name)
			return nil
		}
		
		// Record the error
		lastErr = err
		r.errorHistory = append(r.errorHistory, err)
		
		// Check if error is recoverable
		if migErr, ok := err.(*MigrationError); ok && !migErr.Recoverable {
			// Non-recoverable error, fail immediately
			return err
		}
		
		// Handle based on strategy
		switch r.config.Strategy {
		case FailFast:
			return err
			
		case ContinueOnError:
			// Log and continue
			return nil
			
		case RetryOnError:
			if attempts < maxAttempts {
				// Wait before retry with exponential backoff
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * r.config.RetryBackoff)
			}
		}
	}
	
	// All retries exhausted
	return NewError(TransformError).
		WithOperation("executing with recovery").
		WithContext("operation_name", name).
		WithContext("attempts", attempts).
		WithContext("strategy", r.config.Strategy).
		WithCause(lastErr).
		Build()
}

// Rollback rolls back to the last successful recovery point
func (r *Recoverer) Rollback() error {
	if r.currentPoint < 0 {
		// No recovery points
		return nil
	}
	
	errorList := NewErrorList(10)
	
	// Roll back in reverse order
	for i := r.currentPoint; i >= 0; i-- {
		point := r.points[i]
		
		if point.RestoreFunc != nil {
			if err := point.RestoreFunc(); err != nil {
				errorList.Add(NewError(BackupError).
					WithOperation("rolling back recovery point").
					WithContext("point_name", point.Name).
					WithContext("point_index", i).
					WithCause(err).
					Build())
			}
		}
	}
	
	if errorList.HasErrors() {
		return errorList
	}
	
	return nil
}

// Cleanup cleans up all recovery points
func (r *Recoverer) Cleanup() error {
	errorList := NewErrorList(10)
	
	for _, point := range r.points {
		if point.CleanupFunc != nil {
			// Cleanup functions don't return errors, so we use panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						errorList.Add(NewError(ConfigError).
							WithOperation("cleaning up recovery point").
							WithContext("point_name", point.Name).
							WithContext("panic", r).
							Recoverable().
							Build())
					}
				}()
				point.CleanupFunc()
			}()
		}
	}
	
	if errorList.HasErrors() {
		return errorList
	}
	
	return nil
}

// GetErrorHistory returns all errors encountered
func (r *Recoverer) GetErrorHistory() []error {
	return r.errorHistory
}

// GetSuccessHistory returns all successful operations
func (r *Recoverer) GetSuccessHistory() []string {
	return r.successHistory
}

// GetCurrentPoint returns the current recovery point index
func (r *Recoverer) GetCurrentPoint() int {
	return r.currentPoint
}

// Reset clears the recoverer state
func (r *Recoverer) Reset() {
	r.points = make([]RecoveryPoint, 0)
	r.errorHistory = make([]error, 0)
	r.successHistory = make([]string, 0)
	r.currentPoint = -1
}

// WithRetry is a helper function for simple retry logic
func WithRetry(maxRetries int, delay time.Duration, fn func() error) error {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < maxRetries-1 {
				time.Sleep(delay)
			}
			continue
		}
		return nil
	}
	
	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, lastErr)
}

// RecoverPanic recovers from panics and converts them to errors
func RecoverPanic(operation string) error {
	if r := recover(); r != nil {
		return NewError(TransformError).
			WithOperation(operation).
			WithContext("panic", r).
			WithCause(fmt.Errorf("panic: %v", r)).
			Build()
	}
	return nil
}

// SafeExecute executes a function and recovers from panics
func SafeExecute(operation string, fn func() error) (err error) {
	defer func() {
		if panicErr := RecoverPanic(operation); panicErr != nil {
			err = panicErr
		}
	}()
	
	return fn()
}