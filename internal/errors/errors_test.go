package errors

import (
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestUserError(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			testutil.Equal(t, tt.err.Error(), tt.expected)
		})
	}
}

func TestNewUserError(t *testing.T) {
	t.Parallel()
	err := NewUserError("invalid value: %d", 42)
	testutil.Equal(t, err.Error(), "invalid value: 42")
}

func TestSystemError(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			testutil.Equal(t, tt.err.Error(), tt.expected)
		})
	}
}

func TestSystemErrorUnwrap(t *testing.T) {
	t.Parallel()
	cause := errors.New("underlying error")
	err := SystemError{
		Message: "wrapper",
		Cause:   cause,
	}

	testutil.Equal(t, err.Unwrap(), cause)
	testutil.True(t, errors.Is(err, cause))
}

func TestNewSystemError(t *testing.T) {
	t.Parallel()
	cause := errors.New("network timeout")
	err := NewSystemError("API call failed", cause, true)

	testutil.Equal(t, err.Message, "API call failed")
	testutil.Equal(t, err.Cause, cause)
	testutil.True(t, err.Retryable)
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			testutil.Equal(t, IsRetryable(tt.err), tt.expected)
		})
	}
}
