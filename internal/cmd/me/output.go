package me

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/open-cli-collective/google-readonly/internal/keychain"
	"github.com/open-cli-collective/google-readonly/internal/people"
)

// PeopleClient defines the interface for People client operations used by the me command.
type PeopleClient interface {
	GetMe(ctx context.Context) (*people.Profile, error)
}

// ClientFactory is the function used to create People clients. Override in tests to inject mocks.
var ClientFactory = func(ctx context.Context) (PeopleClient, error) {
	return people.NewClient(ctx)
}

// Extras is the data shown by --extended that doesn't come from People.
type Extras struct {
	GrantedScopes  []string
	TokenExpiry    string
	StorageBackend string
}

// RenderOneLiner writes the canonical `resourceName | displayName | primaryEmail`
// pipe one-liner to w. Empty fields render as "-"; embedded pipes are escaped to
// `\|`; embedded newlines collapse to spaces. Exported so the init wizard can
// render the same line as part of its success message.
func RenderOneLiner(w io.Writer, p *people.Profile) {
	rn := normalizeField(p.ResourceName)
	dn := normalizeField(p.DisplayName)
	em := normalizeField(p.PrimaryEmail)
	_, _ = fmt.Fprintf(w, "%s | %s | %s\n", rn, dn, em)
}

// RenderExtended writes the one-liner plus extended rows to w.
func RenderExtended(w io.Writer, p *people.Profile, e Extras) {
	RenderOneLiner(w, p)
	if len(e.GrantedScopes) > 0 {
		_, _ = fmt.Fprintln(w, "Granted scopes:")
		for _, s := range e.GrantedScopes {
			_, _ = fmt.Fprintf(w, "  - %s\n", s)
		}
	}
	if e.TokenExpiry != "" {
		_, _ = fmt.Fprintf(w, "Token expiry:    %s\n", e.TokenExpiry)
	}
	if e.StorageBackend != "" {
		_, _ = fmt.Fprintf(w, "Storage backend: %s\n", e.StorageBackend)
	}
}

// RenderID writes just the primary email followed by a newline.
func RenderID(w io.Writer, p *people.Profile) {
	em := normalizeField(p.PrimaryEmail)
	_, _ = fmt.Fprintln(w, em)
}

func normalizeField(s string) string {
	if s == "" {
		return "-"
	}
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return s
}

// gatherExtras collects the non-People data shown by --extended. Uses
// OpenNoMigrate: the one-time migration already ran via the People client on
// the main path; this is a read-only display gather and must not re-trigger
// migration or surface a conflict here.
func gatherExtras() Extras {
	var e Extras
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return e
	}
	defer func() { _ = st.Close() }()
	// Best-effort: --extended fields are supplementary; a keyring blip skips
	// them rather than failing `me` (this gather already tolerates an
	// OpenNoMigrate error the same way just above).
	if has, herr := st.HasToken(); herr == nil && has {
		if backend, _ := st.Backend(); backend != "" {
			e.StorageBackend = string(backend)
		}
		if tok, terr := st.Token(); terr == nil && tok != nil && !tok.Expiry.IsZero() {
			e.TokenExpiry = tok.Expiry.Format("2006-01-02T15:04:05Z07:00")
		}
	}
	return e
}
