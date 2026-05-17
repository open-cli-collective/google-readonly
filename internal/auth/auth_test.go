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
	if len(AllScopes) != 7 {
		t.Errorf("got length %d, want %d", len(AllScopes), 7)
	}
	scopeSet := strings.Join(AllScopes, " ")
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/gmail.modify") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/gmail.modify")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/calendar.readonly") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/calendar.readonly")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/calendar.events") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/calendar.events")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/contacts") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/contacts")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/drive.readonly") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/drive.readonly")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/drive.metadata") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/drive.metadata")
	}
	if !strings.Contains(scopeSet, "https://www.googleapis.com/auth/userinfo.profile") {
		t.Errorf("expected AllScopes to contain %q", "https://www.googleapis.com/auth/userinfo.profile")
	}
}

func TestCheckScopesMigration_NoGrantedScopes(t *testing.T) {
	t.Parallel()
	msg := CheckScopesMigration(nil)
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
}

func TestCheckScopesMigration_AllGranted(t *testing.T) {
	t.Parallel()
	msg := CheckScopesMigration(AllScopes)
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
}

func TestCheckScopesMigration_MissingScope(t *testing.T) {
	t.Parallel()
	oldScopes := []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/calendar.readonly",
		"https://www.googleapis.com/auth/contacts.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
	}
	msg := CheckScopesMigration(oldScopes)
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
	if !strings.Contains(msg, "gro init") {
		t.Errorf("expected message to mention 'gro init', got %q", msg)
	}
	if !strings.Contains(msg, "Gmail Modify") {
		t.Errorf("expected message to mention 'Gmail Modify', got %q", msg)
	}
	if !strings.Contains(msg, "Contacts") {
		t.Errorf("expected message to mention 'Contacts', got %q", msg)
	}
}

// TestTokenFromFile was removed: the plaintext token.json fallback no longer
// exists. The token now lives only in the OS keyring via credstore (§1.1 /
// §2.3); legacy token.json is handled one-time by internal/keychain's
// migration and covered by that package's tests.
