package initcmd

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/api/googleapi"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestInitCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "init")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has no-verify flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("no-verify")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Long)
		testutil.Contains(t, cmd.Long, "OAuth")
	})
}

func TestExtractAuthCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "raw code",
			input:    "4/0AQSTgQxyz123",
			expected: "4/0AQSTgQxyz123",
		},
		{
			name:     "localhost URL with code",
			input:    "http://localhost/?code=4/0AQSTgQxyz123&scope=email",
			expected: "4/0AQSTgQxyz123",
		},
		{
			name:     "localhost URL with port",
			input:    "http://localhost:8080/?code=ABC123&scope=email",
			expected: "ABC123",
		},
		{
			name:     "https localhost URL",
			input:    "https://localhost/?code=SecureCode456",
			expected: "SecureCode456",
		},
		{
			name:     "URL without code param",
			input:    "http://localhost/?error=access_denied",
			expected: "",
		},
		{
			name:     "whitespace trimmed",
			input:    "  4/0AQSTgQxyz123  \n",
			expected: "4/0AQSTgQxyz123",
		},
		{
			name:     "whitespace trimmed from URL",
			input:    "  http://localhost/?code=TrimMe  \n",
			expected: "TrimMe",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \n\t  ",
			expected: "",
		},
		{
			name:     "non-localhost URL treated as raw code",
			input:    "http://example.com/?code=NotExtracted",
			expected: "http://example.com/?code=NotExtracted",
		},
		{
			name:     "code with special characters",
			input:    "http://localhost/?code=4/P-abc_123.xyz~456",
			expected: "4/P-abc_123.xyz~456",
		},
		{
			name:     "URL encoded code",
			input:    "http://localhost/?code=4%2F0AQSTgQ",
			expected: "4/0AQSTgQ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAuthCode(tt.input)
			testutil.Equal(t, result, tt.expected)
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "network error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "googleapi 401 error",
			err:      &googleapi.Error{Code: http.StatusUnauthorized, Message: "Invalid Credentials"},
			expected: true,
		},
		{
			name:     "googleapi 403 error",
			err:      &googleapi.Error{Code: http.StatusForbidden, Message: "Access denied"},
			expected: false,
		},
		{
			name:     "googleapi 404 error",
			err:      &googleapi.Error{Code: http.StatusNotFound, Message: "Not found"},
			expected: false,
		},
		{
			name:     "error message with 401 and Invalid Credentials",
			err:      errors.New("googleapi: Error 401: Invalid Credentials"),
			expected: true,
		},
		{
			name:     "error message with 401 and invalid_grant",
			err:      errors.New("oauth2: 401 invalid_grant: Token has been expired"),
			expected: true,
		},
		{
			name:     "error message with Token has been expired or revoked",
			err:      errors.New("401: Token has been expired or revoked"),
			expected: true,
		},
		{
			name:     "error with 401 but no auth keywords",
			err:      errors.New("HTTP 401 response"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAuthError(tt.err)
			testutil.Equal(t, result, tt.expected)
		})
	}
}
