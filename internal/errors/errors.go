// Package errors provides error types for distinguishing between user errors
// and system errors, allowing commands to provide appropriate guidance.
package errors

import (
	"errors"
	"fmt"
)

// UserError represents an error caused by invalid user input or action.
// These errors are actionable - the user can fix them.
type UserError struct {
	Message string
}

func (e UserError) Error() string {
	return e.Message
}

// NewUserError creates a new UserError with the given message.
func NewUserError(format string, args ...any) UserError {
	return UserError{Message: fmt.Sprintf(format, args...)}
}

// SystemError represents an error caused by system issues (API failures,
// network problems, etc). May be temporary and retryable.
type SystemError struct {
	Message   string
	Cause     error
	Retryable bool
}

func (e SystemError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e SystemError) Unwrap() error {
	return e.Cause
}

// NewSystemError creates a new SystemError.
func NewSystemError(message string, cause error, retryable bool) SystemError {
	return SystemError{
		Message:   message,
		Cause:     cause,
		Retryable: retryable,
	}
}

// IsRetryable returns true if the error is a retryable SystemError.
func IsRetryable(err error) bool {
	var sysErr SystemError
	if errors.As(err, &sysErr) {
		return sysErr.Retryable
	}
	return false
}
