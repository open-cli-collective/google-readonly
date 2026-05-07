package people

import (
	"errors"
	"testing"

	"google.golang.org/api/googleapi"
	peopleapi "google.golang.org/api/people/v1"
)

func TestPickPrimaryNamePrefersPrimaryFlag(t *testing.T) {
	t.Parallel()
	p := &peopleapi.Person{
		Names: []*peopleapi.Name{
			{DisplayName: "Other Name", Metadata: &peopleapi.FieldMetadata{Primary: false}},
			{DisplayName: "Primary Name", Metadata: &peopleapi.FieldMetadata{Primary: true}},
		},
	}
	if got := pickPrimaryName(p); got != "Primary Name" {
		t.Fatalf("got %q, want Primary Name", got)
	}
}

func TestPickPrimaryNameFallsBackToFirst(t *testing.T) {
	t.Parallel()
	p := &peopleapi.Person{
		Names: []*peopleapi.Name{
			{DisplayName: "First"},
			{DisplayName: "Second"},
		},
	}
	if got := pickPrimaryName(p); got != "First" {
		t.Fatalf("got %q, want First", got)
	}
}

func TestPickPrimaryNameEmpty(t *testing.T) {
	t.Parallel()
	if got := pickPrimaryName(&peopleapi.Person{}); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
	if got := pickPrimaryName(nil); got != "" {
		t.Fatalf("nil person: got %q, want empty", got)
	}
}

func TestPickPrimaryEmailPrefersPrimaryFlag(t *testing.T) {
	t.Parallel()
	p := &peopleapi.Person{
		EmailAddresses: []*peopleapi.EmailAddress{
			{Value: "alt@example.com", Metadata: &peopleapi.FieldMetadata{Primary: false}},
			{Value: "primary@example.com", Metadata: &peopleapi.FieldMetadata{Primary: true}},
		},
	}
	if got := pickPrimaryEmail(p); got != "primary@example.com" {
		t.Fatalf("got %q, want primary@example.com", got)
	}
}

func TestPickPrimaryEmailFallsBackToFirst(t *testing.T) {
	t.Parallel()
	p := &peopleapi.Person{
		EmailAddresses: []*peopleapi.EmailAddress{
			{Value: "first@example.com"},
			{Value: "second@example.com"},
		},
	}
	if got := pickPrimaryEmail(p); got != "first@example.com" {
		t.Fatalf("got %q, want first@example.com", got)
	}
}

func TestIsInsufficientScopeError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "non-googleapi error", err: errors.New("network blew up"), want: false},
		{name: "401", err: &googleapi.Error{Code: 401}, want: false},
		{
			name: "403 with insufficientPermissions reason",
			err: &googleapi.Error{
				Code:   403,
				Errors: []googleapi.ErrorItem{{Reason: "insufficientPermissions"}},
			},
			want: true,
		},
		{
			name: "403 with ACCESS_TOKEN_SCOPE_INSUFFICIENT reason",
			err: &googleapi.Error{
				Code:   403,
				Errors: []googleapi.ErrorItem{{Reason: "ACCESS_TOKEN_SCOPE_INSUFFICIENT"}},
			},
			want: true,
		},
		{
			name: "403 message mentions insufficient authentication scopes",
			err:  &googleapi.Error{Code: 403, Message: "Request had insufficient authentication scopes."},
			want: true,
		},
		{
			name: "403 PERMISSION_DENIED service disabled — NOT insufficient scope",
			err: &googleapi.Error{
				Code:    403,
				Message: "People API has not been used in project 12345 before or it is disabled.",
				Errors:  []googleapi.ErrorItem{{Reason: "SERVICE_DISABLED"}},
			},
			want: false,
		},
		{
			name: "403 forbidden with no scope marker",
			err: &googleapi.Error{
				Code:    403,
				Message: "The caller does not have permission",
				Errors:  []googleapi.ErrorItem{{Reason: "forbidden"}},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsInsufficientScopeError(tc.err); got != tc.want {
				t.Errorf("IsInsufficientScopeError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestClientStructure(t *testing.T) {
	t.Parallel()
	c := &Client{}
	if c.service != nil {
		t.Fatalf("expected nil service, got %v", c.service)
	}
}
