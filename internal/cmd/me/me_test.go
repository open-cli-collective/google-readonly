package me

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"

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
	err := run(context.Background(), &out, false, false, false)
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
	err := run(context.Background(), &out, false, false, false)
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
	if err := run(context.Background(), &out, false, false, false); err != nil {
		t.Fatal(err)
	}
	want := "people/c1 | Ada | ada@example.com\n"
	if got := out.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
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
	if err := run(context.Background(), &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "ada@example.com\n" {
		t.Fatalf("got %q, want 'ada@example.com\\n'", got)
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
