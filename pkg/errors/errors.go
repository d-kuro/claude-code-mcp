// Package errors provides structured error types and error handling utilities.
package errors

import (
	"errors"
	"fmt"
)

// Wrap creates a new error by wrapping an existing error with additional context.
// This uses fmt.Errorf with %w verb for proper error chain support.
func Wrap(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf is an alias for Wrap that makes the formatting more explicit.
func Wrapf(err error, format string, args ...interface{}) error {
	return Wrap(err, format, args...)
}

// New creates a new error using fmt.Errorf.
// This is a convenience function for creating errors with formatting.
func New(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// Is reports whether any error in err's chain matches target.
// This is a convenience wrapper around errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
// This is a convenience wrapper around errors.As.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Join wraps multiple errors into a single error.
// This is a convenience wrapper around errors.Join (Go 1.20+).
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Error type constants for maintaining backwards compatibility
var (
	ErrValidation    = errors.New("validation error")
	ErrSecurity      = errors.New("security error")
	ErrPermission    = errors.New("permission error")
	ErrConfiguration = errors.New("configuration error")
	ErrExecution     = errors.New("execution error")
	ErrTimeout       = errors.New("timeout error")
	ErrNotFound      = errors.New("not found error")
	ErrInternal      = errors.New("internal error")
)

// Legacy error creation functions for backward compatibility
func Validation(message string) error {
	return fmt.Errorf("VALIDATION_ERROR: %s", message)
}

func ValidationWithDetails(message, details string) error {
	return fmt.Errorf("VALIDATION_ERROR: %s (%s)", message, details)
}

func Security(message string) error {
	return fmt.Errorf("SECURITY_ERROR: %s", message)
}

func SecurityWithDetails(message, details string) error {
	return fmt.Errorf("SECURITY_ERROR: %s (%s)", message, details)
}

func Permission(message string) error {
	return fmt.Errorf("PERMISSION_ERROR: %s", message)
}

func PermissionWithDetails(message, details string) error {
	return fmt.Errorf("PERMISSION_ERROR: %s (%s)", message, details)
}

func Configuration(message string) error {
	return fmt.Errorf("CONFIGURATION_ERROR: %s", message)
}

func ConfigurationWithCause(message string, cause error) error {
	return fmt.Errorf("CONFIGURATION_ERROR: %s: %w", message, cause)
}

func Execution(message string) error {
	return fmt.Errorf("EXECUTION_ERROR: %s", message)
}

func ExecutionWithCause(message string, cause error) error {
	return fmt.Errorf("EXECUTION_ERROR: %s: %w", message, cause)
}

func Timeout(message string) error {
	return fmt.Errorf("TIMEOUT_ERROR: %s", message)
}

func NotFound(message string) error {
	return fmt.Errorf("NOT_FOUND_ERROR: %s", message)
}

func Internal(message string) error {
	return fmt.Errorf("INTERNAL_ERROR: %s", message)
}

func InternalWithCause(message string, cause error) error {
	return fmt.Errorf("INTERNAL_ERROR: %s: %w", message, cause)
}
