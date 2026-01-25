package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserError(t *testing.T) {
	tests := []struct {
		name     string
		err      UserError
		expected string
	}{
		{
			name:     "simple message",
			err:      UserError{Message: "invalid input"},
			expected: "invalid input",
		},
		{
			name:     "empty message",
			err:      UserError{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestNewUserError(t *testing.T) {
	err := NewUserError("invalid value: %d", 42)
	assert.Equal(t, "invalid value: 42", err.Error())
}

func TestSystemError(t *testing.T) {
	tests := []struct {
		name     string
		err      SystemError
		expected string
	}{
		{
			name: "with cause",
			err: SystemError{
				Message:   "API failed",
				Cause:     errors.New("connection refused"),
				Retryable: true,
			},
			expected: "API failed: connection refused",
		},
		{
			name: "without cause",
			err: SystemError{
				Message:   "service unavailable",
				Retryable: true,
			},
			expected: "service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestSystemErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := SystemError{
		Message: "wrapper",
		Cause:   cause,
	}

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestNewSystemError(t *testing.T) {
	cause := errors.New("network timeout")
	err := NewSystemError("API call failed", cause, true)

	assert.Equal(t, "API call failed", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.True(t, err.Retryable)
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable system error",
			err:      SystemError{Message: "timeout", Retryable: true},
			expected: true,
		},
		{
			name:     "non-retryable system error",
			err:      SystemError{Message: "not found", Retryable: false},
			expected: false,
		},
		{
			name:     "user error",
			err:      UserError{Message: "bad input"},
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}
