package me

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"

	"github.com/open-cli-collective/google-readonly/internal/auth"
	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/people"
)

// mockPeopleClient is a stub for the exported PeopleClient interface.
type mockPeopleClient struct {
	GetMeFunc func(ctx context.Context) (*people.Profile, error)
}

func (m *mockPeopleClient) GetMe(ctx context.Context) (*people.Profile, error) {
	if m.GetMeFunc != nil {
		return m.GetMeFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

// withMockClient swaps ClientFactory for the test and restores it on cleanup.
func withMockClient(t *testing.T, c PeopleClient) {
	t.Helper()
	orig := ClientFactory
	ClientFactory = func(_ context.Context) (PeopleClient, error) { return c, nil }
	t.Cleanup(func() { ClientFactory = orig })
}

func TestRenderOneLinerHappyPath(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	RenderOneLiner(&buf, &people.Profile{
		ResourceName: "people/c1234",
		DisplayName:  "Ada Lovelace",
		PrimaryEmail: "ada@example.com",
	})
	want := "people/c1234 | Ada Lovelace | ada@example.com\n"
	if got := buf.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderOneLinerEmptyFields(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	RenderOneLiner(&buf, &people.Profile{ResourceName: "", DisplayName: "", PrimaryEmail: ""})
	want := "- | - | -\n"
	if got := buf.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderOneLinerEscapesPipes(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	RenderOneLiner(&buf, &people.Profile{
		ResourceName: "people/c1",
		DisplayName:  "Has | Pipe",
		PrimaryEmail: "a@b.com",
	})
	if got := buf.String(); !strings.Contains(got, `Has \| Pipe`) {
		t.Fatalf("expected escaped pipe, got %q", got)
	}
}

func TestRenderOneLinerCollapsesNewlines(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	RenderOneLiner(&buf, &people.Profile{
		ResourceName: "people/c1",
		DisplayName:  "First\nSecond",
		PrimaryEmail: "a@b.com",
	})
	got := buf.String()
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("expected exactly one trailing newline, got %q", got)
	}
}

func TestRenderID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	RenderID(&buf, &people.Profile{PrimaryEmail: "ada@example.com"})
	if got := buf.String(); got != "ada@example.com\n" {
		t.Fatalf("got %q, want 'ada@example.com\\n'", got)
	}
}

func TestRenderExtendedIncludesScopesAndExpiry(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	RenderExtended(&buf, &people.Profile{
		ResourceName: "people/c1",
		DisplayName:  "Ada",
		PrimaryEmail: "ada@example.com",
	}, Extras{
		GrantedScopes:  []string{"scope-a", "scope-b"},
		TokenExpiry:    "2030-01-01T00:00:00Z",
		StorageBackend: "keychain",
	})
	got := buf.String()
	for _, want := range []string{"people/c1", "ada@example.com", "scope-a", "scope-b", "2030-01-01T00:00:00Z", "keychain"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderJSONShapes(t *testing.T) {
	t.Parallel()
	p := &people.Profile{
		ResourceName: "people/c1",
		DisplayName:  "Ada",
		PrimaryEmail: "ada@example.com",
	}
	extras := Extras{GrantedScopes: []string{"scope-a"}, TokenExpiry: "expiry-x", StorageBackend: "keychain"}

	t.Run("default", func(t *testing.T) {
		var buf bytes.Buffer
		if err := RenderJSON(&buf, p, Extras{}, false, false); err != nil {
			t.Fatal(err)
		}
		var got jsonOneLiner
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.ResourceName != "people/c1" || got.DisplayName != "Ada" || got.PrimaryEmail != "ada@example.com" {
			t.Errorf("unexpected: %+v", got)
		}
	})

	t.Run("--id", func(t *testing.T) {
		var buf bytes.Buffer
		if err := RenderJSON(&buf, p, Extras{}, true, false); err != nil {
			t.Fatal(err)
		}
		var got jsonIDOnly
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.PrimaryEmail != "ada@example.com" {
			t.Errorf("unexpected: %+v", got)
		}
		// Make sure the other fields are absent.
		if strings.Contains(buf.String(), "resourceName") {
			t.Errorf("--id JSON should not include resourceName, got %s", buf.String())
		}
	})

	t.Run("--extended", func(t *testing.T) {
		var buf bytes.Buffer
		if err := RenderJSON(&buf, p, extras, false, true); err != nil {
			t.Fatal(err)
		}
		var got jsonExtended
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.ResourceName != "people/c1" || got.PrimaryEmail != "ada@example.com" {
			t.Errorf("unexpected: %+v", got)
		}
		if len(got.GrantedScopes) != 1 || got.GrantedScopes[0] != "scope-a" {
			t.Errorf("expected scope-a, got %+v", got.GrantedScopes)
		}
		if got.TokenExpiry != "expiry-x" || got.StorageBackend != "keychain" {
			t.Errorf("unexpected: %+v", got)
		}
	})
}

func TestRunInsufficientScopeRemap(t *testing.T) {
	// Not Parallel: mutates package-global ClientFactory.
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return nil, &googleapi.Error{
				Code:   403,
				Errors: []googleapi.ErrorItem{{Reason: "ACCESS_TOKEN_SCOPE_INSUFFICIENT"}},
			}
		},
	})
	var out bytes.Buffer
	err := run(context.Background(), &out, &bytes.Buffer{}, false, false, false)
	if !errors.Is(err, errReauth) {
		t.Fatalf("expected errReauth, got %v", err)
	}
}

func TestRunServiceDisabledNotRemapped(t *testing.T) {
	// Not Parallel: mutates package-global ClientFactory.
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return nil, &googleapi.Error{
				Code:    403,
				Message: "People API has not been used in project 12345 before or it is disabled.",
				Errors:  []googleapi.ErrorItem{{Reason: "SERVICE_DISABLED"}},
			}
		},
	})
	var out bytes.Buffer
	err := run(context.Background(), &out, &bytes.Buffer{}, false, false, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, errReauth) {
		t.Fatalf("service-disabled 403 should NOT be remapped to errReauth, got %v", err)
	}
	if !strings.Contains(err.Error(), "people API") || !strings.Contains(err.Error(), "console.cloud.google.com") {
		t.Errorf("expected guidance toward Cloud Console, got %v", err)
	}
}

func TestRunDefaultPipeOneLiner(t *testing.T) {
	// Not Parallel: mutates package-global ClientFactory.
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return &people.Profile{
				ResourceName: "people/c1",
				DisplayName:  "Ada",
				PrimaryEmail: "ada@example.com",
			}, nil
		},
	})
	var out bytes.Buffer
	if err := run(context.Background(), &out, &bytes.Buffer{}, false, false, false); err != nil {
		t.Fatal(err)
	}
	want := "people/c1 | Ada | ada@example.com\n"
	if got := out.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestRunExtendedJSONEndToEnd ensures that --extended + --json flow through
// run() correctly: the renderer is tested in isolation by TestRenderJSONShapes,
// but only this test catches a regression where gatherExtras is skipped when
// jsonOutput is true.
func TestRunExtendedJSONEndToEnd(t *testing.T) {
	// Not Parallel: mutates package-global ClientFactory + env.
	withConfigDir(t)
	// Seed config with a known scope record so we can assert the JSON
	// surface reflects the recorded scopes, not auth.AllScopes (which
	// would overstate consent).
	if err := config.SaveConfig(&config.Config{
		CacheTTLHours: 24,
		GrantedScopes: auth.AllScopes,
	}); err != nil {
		t.Fatal(err)
	}

	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return &people.Profile{
				ResourceName: "people/c1",
				DisplayName:  "Ada",
				PrimaryEmail: "ada@example.com",
			}, nil
		},
	})
	var out, errOut bytes.Buffer
	if err := run(context.Background(), &out, &errOut, false /*idOnly*/, true /*extended*/, true /*json*/); err != nil {
		t.Fatalf("run: %v", err)
	}
	var got jsonExtended
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out.String())
	}
	if got.PrimaryEmail != "ada@example.com" {
		t.Errorf("expected ada@example.com, got %+v", got)
	}
	// Recorded scopes should propagate through to extended output. This
	// guards against a regression where gatherExtras is skipped under
	// jsonOutput.
	if len(got.GrantedScopes) != len(auth.AllScopes) {
		t.Errorf("expected %d scopes, got %d: %+v", len(auth.AllScopes), len(got.GrantedScopes), got.GrantedScopes)
	}
}

func TestRunIDOnlyEmitsEmail(t *testing.T) {
	// Not Parallel: mutates package-global ClientFactory.
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return &people.Profile{
				ResourceName: "people/c1",
				DisplayName:  "Ada",
				PrimaryEmail: "ada@example.com",
			}, nil
		},
	})
	var out bytes.Buffer
	if err := run(context.Background(), &out, &bytes.Buffer{}, true, false, false); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "ada@example.com\n" {
		t.Fatalf("got %q, want 'ada@example.com\\n'", got)
	}
}

// TestNewCommandSilencesCobraErrorOnReauth verifies that running the cobra
// command end-to-end with a reauth-required scenario doesn't produce the
// double "Error: re-authentication required" line cobra would normally add.
func TestNewCommandSilencesCobraErrorOnReauth(t *testing.T) {
	// Not Parallel: mutates package-global ClientFactory + env.
	withConfigDir(t)
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return nil, &googleapi.Error{
				Code:   403,
				Errors: []googleapi.ErrorItem{{Reason: "ACCESS_TOKEN_SCOPE_INSUFFICIENT"}},
			}
		},
	})
	cmd := NewCommand()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if !errors.Is(err, errReauth) {
		t.Fatalf("expected errReauth from Execute, got %v", err)
	}
	// Cobra would normally write "Error: re-authentication required" when
	// returning a non-nil error. SilenceErrors should suppress that.
	if strings.Contains(errOut.String(), "Error: re-authentication required") {
		t.Errorf("expected SilenceErrors to suppress cobra's error prefix, but stderr was:\n%s", errOut.String())
	}
	// We should still see the actionable guidance.
	if !strings.Contains(errOut.String(), "gro init") {
		t.Errorf("expected actionable 'gro init' guidance, got:\n%s", errOut.String())
	}
}

func TestNewCommandHasJSONFlagWithShorthand(t *testing.T) {
	t.Parallel()
	cmd := NewCommand()
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("expected --json flag")
	}
	if flag.Shorthand != "j" {
		t.Fatalf("expected -j shorthand, got %q", flag.Shorthand)
	}
}

func TestNewCommandIDExtendedMutuallyExclusive(t *testing.T) {
	t.Parallel()
	cmd := NewCommand()
	cmd.SetArgs([]string{"--id", "--extended"})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected mutual-exclusion error")
	}
	// Cobra phrases the error as "if any flags in the group [id extended] are set
	// none of the others can be" — assert the field names rather than the wording.
	if !strings.Contains(err.Error(), "id") || !strings.Contains(err.Error(), "extended") {
		t.Fatalf("expected error to mention both flags, got %v", err)
	}
}

// withConfigDir points config.LoadConfig at a clean tempdir so stale-scope
// state can be injected per-test via real config.json.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestRunStaleRecordedScopesTriggersReauthMessage(t *testing.T) {
	// Not Parallel: mutates env + ClientFactory.
	withConfigDir(t)
	// Write a config.json with a deliberately incomplete scopes list — only
	// gmail.modify, missing People/Drive/etc. CheckScopesMigration should fire.
	cfg := &config.Config{
		CacheTTLHours: 24,
		GrantedScopes: []string{"https://www.googleapis.com/auth/gmail.modify"},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			t.Fatal("client should not have been called when scopes are stale")
			return nil, nil
		},
	})
	var out, errOut bytes.Buffer
	err := run(context.Background(), &out, &errOut, false, false, false)
	if !errors.Is(err, errReauth) {
		t.Fatalf("expected errReauth, got %v", err)
	}
	if !strings.Contains(errOut.String(), "gro init") {
		t.Errorf("expected 'gro init' guidance in stderr, got %q", errOut.String())
	}
}

func TestRunMissingConfigDoesNotShortCircuit(t *testing.T) {
	// Not Parallel: mutates env + ClientFactory.
	withConfigDir(t)
	// No config.json on disk: CheckScopesMigration receives nil scopes and
	// short-circuits to empty. The test asserts we do NOT spuriously fire
	// errReauth in that case (it's a regression for the "missing config"
	// edge Codex flagged).
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return &people.Profile{ResourceName: "people/c1", DisplayName: "Ada", PrimaryEmail: "ada@example.com"}, nil
		},
	})
	var out, errOut bytes.Buffer
	if err := run(context.Background(), &out, &errOut, false, false, false); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.Contains(out.String(), "ada@example.com") {
		t.Errorf("expected one-liner output, got %q", out.String())
	}
}

func TestRunEmptyGrantedScopesDoesNotShortCircuit(t *testing.T) {
	// Not Parallel: mutates env + ClientFactory.
	withConfigDir(t)
	// Config exists but granted_scopes is explicitly empty — same semantics
	// as missing-config per CheckScopesMigration's len-0 early return.
	cfg := &config.Config{CacheTTLHours: 24, GrantedScopes: []string{}}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	withMockClient(t, &mockPeopleClient{
		GetMeFunc: func(_ context.Context) (*people.Profile, error) {
			return &people.Profile{ResourceName: "people/c1", DisplayName: "Ada", PrimaryEmail: "ada@example.com"}, nil
		},
	})
	var out, errOut bytes.Buffer
	if err := run(context.Background(), &out, &errOut, false, false, false); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
