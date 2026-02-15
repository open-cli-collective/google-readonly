package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

// TestDeprecatedWrappers verifies that auth package wrappers delegate to config package
func TestDeprecatedWrappers(t *testing.T) {
	t.Run("GetConfigDir delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authDir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configDir, err := config.GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if authDir != configDir {
			t.Errorf("got %v, want %v", authDir, configDir)
		}
	})

	t.Run("GetCredentialsPath delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authPath, err := GetCredentialsPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath, err := config.GetCredentialsPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if authPath != configPath {
			t.Errorf("got %v, want %v", authPath, configPath)
		}
	})

	t.Run("GetTokenPath delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authPath, err := GetTokenPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath, err := config.GetTokenPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if authPath != configPath {
			t.Errorf("got %v, want %v", authPath, configPath)
		}
	})

	t.Run("ShortenPath delegates to config package", func(t *testing.T) {
		t.Parallel()
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testPath := filepath.Join(home, ".config", "test")

		authResult := ShortenPath(testPath)
		configResult := config.ShortenPath(testPath)

		if authResult != configResult {
			t.Errorf("got %v, want %v", authResult, configResult)
		}
	})

	t.Run("Constants match config package", func(t *testing.T) {
		t.Parallel()
		if ConfigDirName != config.DirName {
			t.Errorf("got %v, want %v", ConfigDirName, config.DirName)
		}
		if CredentialsFile != config.CredentialsFile {
			t.Errorf("got %v, want %v", CredentialsFile, config.CredentialsFile)
		}
		if TokenFile != config.TokenFile {
			t.Errorf("got %v, want %v", TokenFile, config.TokenFile)
		}
	})
}

func TestAllScopes(t *testing.T) {
	t.Parallel()
	if len(AllScopes) != 4 {
		t.Errorf("got length %d, want %d", len(AllScopes), 4)
	}
	scopeSet := strings.Join(AllScopes, " ")
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/gmail.readonly") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/gmail.readonly")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/calendar.readonly") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/calendar.readonly")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/contacts.readonly") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/contacts.readonly")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/drive.readonly") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/drive.readonly")
	}
}

func TestTokenFromFile(t *testing.T) {
	t.Parallel()
	t.Run("reads valid token file", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "token.json")

		tokenData := `{
			"access_token": "test-access-token",
			"token_type": "Bearer",
			"refresh_token": "test-refresh-token",
			"expiry": "2024-01-01T00:00:00Z"
		}`
		err := os.WriteFile(tokenPath, []byte(tokenData), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		token, err := tokenFromFile(tokenPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token.AccessToken != "test-access-token" {
			t.Errorf("got %v, want %v", token.AccessToken, "test-access-token")
		}
		if token.TokenType != "Bearer" {
			t.Errorf("got %v, want %v", token.TokenType, "Bearer")
		}
		if token.RefreshToken != "test-refresh-token" {
			t.Errorf("got %v, want %v", token.RefreshToken, "test-refresh-token")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()
		_, err := tokenFromFile("/nonexistent/token.json")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "token.json")

		err := os.WriteFile(tokenPath, []byte("not valid json"), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = tokenFromFile(tokenPath)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
