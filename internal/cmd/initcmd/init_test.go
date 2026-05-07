package initcmd

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
	"github.com/open-cli-collective/google-readonly/internal/people"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
	"github.com/open-cli-collective/google-readonly/internal/view"
)

const validOAuthJSON = `{
  "installed": {
    "client_id": "1234.apps.googleusercontent.com",
    "project_id": "test",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
    "client_secret": "shh",
    "redirect_uris": ["http://localhost"]
  }
}`

func TestInitCommand(t *testing.T) {
	t.Parallel()
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		t.Parallel()
		testutil.Equal(t, cmd.Use, "init")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		t.Parallel()
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)
		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has expected flags", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"no-verify", "no-browser", "credentials-file"} {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("missing flag: %s", name)
			}
		}
	})

	t.Run("Long mentions People API", func(t *testing.T) {
		t.Parallel()
		testutil.Contains(t, cmd.Long, "People API")
		testutil.Contains(t, cmd.Long, "Gmail")
		testutil.Contains(t, cmd.Long, "Calendar")
		testutil.Contains(t, cmd.Long, "Drive")
	})
}

func TestExtractAuthCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"raw code", "4/0AQSTgQxyz123", "4/0AQSTgQxyz123"},
		{"localhost URL with code", "http://localhost/?code=4/0AQSTgQxyz123&scope=email", "4/0AQSTgQxyz123"},
		{"localhost URL with port", "http://localhost:8080/?code=ABC123&scope=email", "ABC123"},
		{"https localhost URL", "https://localhost/?code=SecureCode456", "SecureCode456"},
		{"URL without code param", "http://localhost/?error=access_denied", ""},
		{"whitespace trimmed", "  4/0AQSTgQxyz123  \n", "4/0AQSTgQxyz123"},
		{"whitespace trimmed from URL", "  http://localhost/?code=TrimMe  \n", "TrimMe"},
		{"empty input", "", ""},
		{"whitespace only", "   \n\t  ", ""},
		{"non-localhost URL treated as raw code", "http://example.com/?code=NotExtracted", "http://example.com/?code=NotExtracted"},
		{"code with special characters", "http://localhost/?code=4/P-abc_123.xyz~456", "4/P-abc_123.xyz~456"},
		{"URL encoded code", "http://localhost/?code=4%2F0AQSTgQ", "4/0AQSTgQ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testutil.Equal(t, extractAuthCode(tt.input), tt.expected)
		})
	}
}

func TestIsAuthError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"generic error", errors.New("something went wrong"), false},
		{"network error", errors.New("connection refused"), false},
		{"googleapi 401", &googleapi.Error{Code: http.StatusUnauthorized, Message: "Invalid Credentials"}, true},
		{"googleapi 403", &googleapi.Error{Code: http.StatusForbidden, Message: "Access denied"}, false},
		{"googleapi 404", &googleapi.Error{Code: http.StatusNotFound, Message: "Not found"}, false},
		{"text 401 + Invalid Credentials", errors.New("googleapi: Error 401: Invalid Credentials"), true},
		{"text 401 + invalid_grant", errors.New("oauth2: 401 invalid_grant: Token has been expired"), true},
		{"text Token has been expired or revoked", errors.New("401: Token has been expired or revoked"), true},
		{"text 401 alone", errors.New("HTTP 401 response"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testutil.Equal(t, isAuthError(tt.err), tt.expected)
		})
	}
}

func TestValidateOAuthJSONRejectsGarbage(t *testing.T) {
	t.Parallel()
	if err := validateOAuthJSON("not json"); err == nil {
		t.Fatal("expected error for garbage")
	}
	if err := validateOAuthJSON(""); err == nil {
		t.Fatal("expected error for empty")
	}
	if err := validateOAuthJSON(validOAuthJSON); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

// stubPrompter records calls and returns canned values.
type stubPrompter struct {
	credChoice  string
	pasteJSON   string
	filePath    string
	openBrowser bool
	redirectURL string
	reauth      bool
	ttl         int

	pasteJSONErr error
	filePathErr  error

	calls []string
}

func (s *stubPrompter) SelectCredSource(_ bool) (string, error) {
	s.calls = append(s.calls, "select")
	return s.credChoice, nil
}
func (s *stubPrompter) PasteJSON() (string, error) {
	s.calls = append(s.calls, "paste")
	return s.pasteJSON, s.pasteJSONErr
}
func (s *stubPrompter) FilePath() (string, error) {
	s.calls = append(s.calls, "file")
	return s.filePath, s.filePathErr
}
func (s *stubPrompter) ConfirmOpenBrowser() (bool, error) {
	s.calls = append(s.calls, "browser")
	return s.openBrowser, nil
}
func (s *stubPrompter) PasteRedirectURL() (string, error) {
	s.calls = append(s.calls, "redirect")
	return s.redirectURL, nil
}
func (s *stubPrompter) ConfirmReauth() (bool, error) {
	s.calls = append(s.calls, "reauth")
	return s.reauth, nil
}
func (s *stubPrompter) AskCacheTTL(d int) (int, error) {
	s.calls = append(s.calls, "ttl")
	if s.ttl == 0 {
		return d, nil
	}
	return s.ttl, nil
}

// fakeFS captures filesystem interactions across writeCredentials and Stat.
type fakeFS struct {
	files map[string][]byte
	perms map[string]os.FileMode
}

func newFakeFS() *fakeFS {
	return &fakeFS{files: map[string][]byte{}, perms: map[string]os.FileMode{}}
}

func (f *fakeFS) ReadFile(p string) ([]byte, error) {
	if b, ok := f.files[p]; ok {
		return b, nil
	}
	return nil, os.ErrNotExist
}
func (f *fakeFS) WriteFile(p string, data []byte, perm os.FileMode) error {
	f.files[p] = data
	f.perms[p] = perm
	return nil
}
func (f *fakeFS) Chmod(p string, perm os.FileMode) error {
	if _, ok := f.files[p]; !ok {
		return os.ErrNotExist
	}
	f.perms[p] = perm
	return nil
}
func (f *fakeFS) Stat(p string) (os.FileInfo, error) {
	if _, ok := f.files[p]; ok {
		return fakeFileInfo{name: p}, nil
	}
	return nil, os.ErrNotExist
}

type fakeFileInfo struct{ name string }

func (f fakeFileInfo) Name() string       { return filepath.Base(f.name) }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func baseDeps(t *testing.T, fs *fakeFS) initDeps {
	t.Helper()
	credPath := filepath.Join(t.TempDir(), "credentials.json")
	configPath := filepath.Join(filepath.Dir(credPath), "config.json")
	cfgPtr := &config.Config{}

	return initDeps{
		View:               view.NewWithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
		GetCredentialsPath: func() (string, error) { return credPath, nil },
		ReadFile: func(p string) ([]byte, error) {
			if b, err := fs.ReadFile(p); err == nil {
				return b, nil
			}
			return os.ReadFile(p)
		},
		WriteFile: fs.WriteFile,
		Chmod:     fs.Chmod,
		Stat: func(p string) (os.FileInfo, error) {
			// Real os.Stat for filesystem under TempDir; fakeFS only knows
			// what we put in it.
			info, err := os.Stat(p)
			if err == nil {
				return info, nil
			}
			return fs.Stat(p)
		},
		ClipboardSupported: func() bool { return false },
		ClipboardReadAll:   func() (string, error) { return "", errors.New("disabled") },
		OpenBrowser:        func(_ string) error { return nil },
		HasStoredToken:     func() bool { return false },
		SetToken:           func(_ *oauth2.Token) error { return nil },
		DeleteToken:        func() error { return nil },
		GetStorageBackend:  func() keychain.StorageBackend { return "test" },
		ExchangeAuthCode: func(_ context.Context, _ *oauth2.Config, _ string) (*oauth2.Token, error) {
			return &oauth2.Token{AccessToken: "tok"}, nil
		},
		GetOAuthConfig: func() (*oauth2.Config, error) { return &oauth2.Config{}, nil },
		GmailVerify:    func(_ context.Context) (string, error) { return "ada@example.com", nil },
		PeopleGetMe: func(_ context.Context) (*people.Profile, error) {
			return &people.Profile{ResourceName: "people/c1", DisplayName: "Ada", PrimaryEmail: "ada@example.com"}, nil
		},
		LoadConfig:    func() (*config.Config, error) { return cfgPtr, nil },
		SaveConfig:    func(c *config.Config) error { *cfgPtr = *c; return nil },
		GetConfigPath: func() (string, error) { return configPath, nil },
	}
}

func TestEnsureCredentialsFlagFile(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "downloaded.json")
	if err := os.WriteFile(src, []byte(validOAuthJSON), 0644); err != nil {
		t.Fatal(err)
	}
	dst, _ := d.GetCredentialsPath()

	d.Prompter = &stubPrompter{}
	if err := ensureCredentials(d, &initOptions{credentialsFile: src}, dst); err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	got, err := fs.ReadFile(dst)
	if err != nil {
		t.Fatalf("dst not written: %v", err)
	}
	if string(got) != validOAuthJSON {
		t.Errorf("dst content mismatch")
	}
	if perm := fs.perms[dst]; perm != 0600 {
		t.Errorf("expected perms 0600, got %o", perm)
	}
}

func TestEnsureCredentialsRejectsBadJSON(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "bad.json")
	if err := os.WriteFile(src, []byte("garbage"), 0644); err != nil {
		t.Fatal(err)
	}
	dst, _ := d.GetCredentialsPath()
	d.Prompter = &stubPrompter{}
	if err := ensureCredentials(d, &initOptions{credentialsFile: src}, dst); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEnsureCredentialsClipboardWizard(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	d.ClipboardSupported = func() bool { return true }
	d.ClipboardReadAll = func() (string, error) { return validOAuthJSON, nil }
	dst, _ := d.GetCredentialsPath()
	d.Prompter = &stubPrompter{credChoice: "clipboard"}
	if err := ensureCredentials(d, &initOptions{}, dst); err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if _, err := fs.ReadFile(dst); err != nil {
		t.Fatalf("dst not written: %v", err)
	}
}

func TestEnsureCredentialsPasteWizard(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	dst, _ := d.GetCredentialsPath()
	d.Prompter = &stubPrompter{credChoice: "paste", pasteJSON: validOAuthJSON}
	if err := ensureCredentials(d, &initOptions{}, dst); err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if _, err := fs.ReadFile(dst); err != nil {
		t.Fatalf("dst not written: %v", err)
	}
}

func TestEnsureCredentialsTightensPermsOnOverwrite(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	dst, _ := d.GetCredentialsPath()
	// Pre-existing 0644 file.
	if err := os.WriteFile(dst, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dst)

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "downloaded.json")
	if err := os.WriteFile(src, []byte(validOAuthJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Use real FS for write/chmod since the test asserts on disk perms.
	d.WriteFile = os.WriteFile
	d.Chmod = os.Chmod
	d.ReadFile = os.ReadFile
	d.Prompter = &stubPrompter{}
	if err := ensureCredentials(d, &initOptions{credentialsFile: src}, dst); err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 after overwrite, got %o", info.Mode().Perm())
	}
}

func TestRunWithFreshSetupSavesScopesAndAsksTTL(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "downloaded.json")
	if err := os.WriteFile(src, []byte(validOAuthJSON), 0644); err != nil {
		t.Fatal(err)
	}
	cfgSeen := []*config.Config{}
	d.SaveConfig = func(c *config.Config) error {
		cp := *c
		cfgSeen = append(cfgSeen, &cp)
		return nil
	}
	stub := &stubPrompter{credChoice: "paste", pasteJSON: validOAuthJSON, redirectURL: "http://localhost/?code=ABC", ttl: 12}
	d.Prompter = stub

	if err := runWith(context.Background(), d, &initOptions{credentialsFile: src}); err != nil {
		t.Fatalf("runWith: %v", err)
	}

	// The TTL prompt should have been called even though we save scopes
	// (which creates config.json in real life). The snapshot-before-save
	// fix means it is gated on the pre-run state, not the live one.
	if !contains(stub.calls, "ttl") {
		t.Errorf("expected TTL prompt, calls=%v", stub.calls)
	}
	if len(cfgSeen) < 2 {
		t.Fatalf("expected at least two config saves (scopes, then TTL), got %d", len(cfgSeen))
	}
	if cfgSeen[len(cfgSeen)-1].CacheTTLHours != 12 {
		t.Errorf("expected CacheTTLHours=12, got %d", cfgSeen[len(cfgSeen)-1].CacheTTLHours)
	}
}

func TestRunWithExpiredTokenPromptsReauth(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	d.HasStoredToken = func() bool { return true }
	calls := 0
	d.GmailVerify = func(_ context.Context) (string, error) {
		calls++
		if calls == 1 {
			return "", &googleapi.Error{Code: http.StatusUnauthorized, Message: "Invalid Credentials"}
		}
		return "ada@example.com", nil
	}
	deleteCalled := false
	d.DeleteToken = func() error { deleteCalled = true; return nil }

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "downloaded.json")
	if err := os.WriteFile(src, []byte(validOAuthJSON), 0644); err != nil {
		t.Fatal(err)
	}

	stub := &stubPrompter{credChoice: "paste", pasteJSON: validOAuthJSON, redirectURL: "http://localhost/?code=ABC", reauth: true, ttl: 6}
	d.Prompter = stub

	if err := runWith(context.Background(), d, &initOptions{credentialsFile: src}); err != nil {
		t.Fatalf("runWith: %v", err)
	}
	if !deleteCalled {
		t.Errorf("expected DeleteToken to be called after re-auth confirm")
	}
	if !contains(stub.calls, "reauth") {
		t.Errorf("expected reauth prompt")
	}
}

func TestRunWithBadRedirectURLReturnsError(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "downloaded.json")
	if err := os.WriteFile(src, []byte(validOAuthJSON), 0644); err != nil {
		t.Fatal(err)
	}
	d.Prompter = &stubPrompter{credChoice: "paste", pasteJSON: validOAuthJSON, redirectURL: "http://localhost/?error=denied"}
	err := runWith(context.Background(), d, &initOptions{credentialsFile: src})
	if err == nil || !strings.Contains(err.Error(), "no authorization code") {
		t.Fatalf("expected 'no authorization code' error, got %v", err)
	}
}

// TestRunWithRecordedStaleScopesReauths covers the loud-and-early branch in
// tryExistingToken that fires before any API call: when config.json records
// scopes missing from auth.AllScopes (typical of users who upgraded gro
// before adding new scopes), we prompt for re-auth without ever calling
// Gmail/People.
func TestRunWithRecordedStaleScopesReauths(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)

	credPath, _ := d.GetCredentialsPath()
	if err := os.WriteFile(credPath, []byte(validOAuthJSON), 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(credPath)

	d.HasStoredToken = func() bool { return true }
	// Recorded scopes are deliberately incomplete — only gmail.modify.
	d.LoadConfig = func() (*config.Config, error) {
		return &config.Config{
			CacheTTLHours: 24,
			GrantedScopes: []string{"https://www.googleapis.com/auth/gmail.modify"},
		}, nil
	}
	// Track call order: the recorded-stale-scope branch must call DeleteToken
	// BEFORE any GmailVerify / PeopleGetMe (which only happen in the post-
	// OAuth verify path).
	var order []string
	d.DeleteToken = func() error { order = append(order, "delete"); return nil }
	d.GmailVerify = func(_ context.Context) (string, error) {
		order = append(order, "gmail")
		return "ada@example.com", nil
	}
	d.PeopleGetMe = func(_ context.Context) (*people.Profile, error) {
		order = append(order, "people")
		return &people.Profile{ResourceName: "people/c1", DisplayName: "Ada", PrimaryEmail: "ada@example.com"}, nil
	}
	d.ExchangeAuthCode = func(_ context.Context, _ *oauth2.Config, _ string) (*oauth2.Token, error) {
		order = append(order, "exchange")
		return &oauth2.Token{AccessToken: "tok"}, nil
	}

	stub := &stubPrompter{redirectURL: "http://localhost/?code=ABC", reauth: true, ttl: 24}
	d.Prompter = stub

	if err := runWith(context.Background(), d, &initOptions{}); err != nil {
		t.Fatalf("runWith: %v", err)
	}
	// First op must be delete (proves the loud-and-early stale-scope branch
	// fired BEFORE Gmail or People were ever called on the stale token).
	if len(order) == 0 || order[0] != "delete" {
		t.Errorf("expected first op to be delete, got order=%v", order)
	}
	if !contains(stub.calls, "redirect") {
		t.Errorf("expected fresh OAuth flow after re-auth, calls=%v", stub.calls)
	}
}

// TestRunWithExistingTokenStaleScopeReauths covers the #107 stale-scope
// remediation loop: a token that's Gmail-valid but People-insufficient
// must NOT be accepted as-is by `gro init`, otherwise users are stuck in
// "gro me says run gro init / gro init says you're fine" forever.
func TestRunWithExistingTokenStaleScopeReauths(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)

	// Pre-populate credentials.json so we don't enter the wizard.
	credPath, _ := d.GetCredentialsPath()
	if err := os.WriteFile(credPath, []byte(validOAuthJSON), 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(credPath)

	d.HasStoredToken = func() bool { return true }
	d.GmailVerify = func(_ context.Context) (string, error) { return "ada@example.com", nil }
	peopleCalls := 0
	d.PeopleGetMe = func(_ context.Context) (*people.Profile, error) {
		peopleCalls++
		if peopleCalls == 1 {
			return nil, &googleapi.Error{
				Code:   403,
				Errors: []googleapi.ErrorItem{{Reason: "ACCESS_TOKEN_SCOPE_INSUFFICIENT"}},
			}
		}
		return &people.Profile{ResourceName: "people/c1", DisplayName: "Ada", PrimaryEmail: "ada@example.com"}, nil
	}
	deleteCalled := false
	d.DeleteToken = func() error { deleteCalled = true; return nil }

	stub := &stubPrompter{redirectURL: "http://localhost/?code=ABC", reauth: true}
	d.Prompter = stub

	if err := runWith(context.Background(), d, &initOptions{}); err != nil {
		t.Fatalf("runWith: %v", err)
	}
	if !deleteCalled {
		t.Errorf("expected DeleteToken to be called when token is People-insufficient")
	}
	if !contains(stub.calls, "redirect") {
		t.Errorf("expected fresh OAuth flow after re-auth, calls=%v", stub.calls)
	}
}

// TestRunWithExistingTokenNoVerifySkipsAPI covers the --no-verify regression:
// an existing token must be accepted without any API calls.
func TestRunWithExistingTokenNoVerifySkipsAPI(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)

	credPath, _ := d.GetCredentialsPath()
	if err := os.WriteFile(credPath, []byte(validOAuthJSON), 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(credPath)

	d.HasStoredToken = func() bool { return true }
	d.GmailVerify = func(_ context.Context) (string, error) {
		t.Fatal("GmailVerify should not be called with --no-verify")
		return "", nil
	}
	d.PeopleGetMe = func(_ context.Context) (*people.Profile, error) {
		t.Fatal("PeopleGetMe should not be called with --no-verify")
		return nil, nil
	}
	d.Prompter = &stubPrompter{}

	if err := runWith(context.Background(), d, &initOptions{noVerify: true}); err != nil {
		t.Fatalf("runWith: %v", err)
	}
}

// TestRunWithFreshSetupPeopleFailureIsFatal covers the "setup complete is a
// lie" regression: if People verify fails post-OAuth, init must surface the
// error rather than printing "Setup complete" + suggesting `gro me`.
func TestRunWithFreshSetupPeopleFailureIsFatal(t *testing.T) {
	t.Parallel()
	fs := newFakeFS()
	d := baseDeps(t, fs)
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "downloaded.json")
	if err := os.WriteFile(src, []byte(validOAuthJSON), 0644); err != nil {
		t.Fatal(err)
	}
	d.PeopleGetMe = func(_ context.Context) (*people.Profile, error) {
		return nil, &googleapi.Error{Code: 403, Message: "People API has not been used in project before"}
	}
	stub := &stubPrompter{credChoice: "paste", pasteJSON: validOAuthJSON, redirectURL: "http://localhost/?code=ABC", ttl: 12}
	d.Prompter = stub

	err := runWith(context.Background(), d, &initOptions{credentialsFile: src})
	if err == nil {
		t.Fatal("expected People failure to be fatal")
	}
	if !strings.Contains(err.Error(), "People") && !strings.Contains(err.Error(), "people") {
		t.Errorf("expected 'people' in error, got %v", err)
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
