package keychain

import (
	"errors"
	"testing"

	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/credtest"
)

// fakeBase returns toks[0], then toks[1], ... one per Token() call (last one
// sticks), simulating oauth2's transparent refresh.
type fakeBase struct {
	toks []*oauth2.Token
	n    int
}

func (f *fakeBase) Token() (*oauth2.Token, error) {
	t := f.toks[f.n]
	if f.n < len(f.toks)-1 {
		f.n++
	}
	return t, nil
}

// TestPersistentTokenSource_RefreshPersistsToCapturedRef is the pinned test
// for the one sanctioned non-ingress keyring write (standard §ix): a refresh
// after the source is built must land under the captured ref/key, create no
// extra keys, and not change the ref.
func TestPersistentTokenSource_RefreshPersistsToCapturedRef(t *testing.T) {
	credtest.Setup(t)
	const ref = "google-readonly/work" // deliberately NOT the default ref

	persistCalls := 0
	persist := func(tok *oauth2.Token) error {
		persistCalls++
		st, err := openWith(&config.Config{CredentialRef: ref}, false, false)
		if err != nil {
			return err
		}
		defer func() { _ = st.Close() }()
		return st.SetToken(tok)
	}

	initial := &oauth2.Token{AccessToken: "OLD", RefreshToken: "R"}
	refreshed := &oauth2.Token{AccessToken: "NEW", RefreshToken: "R"}
	ts := &PersistentTokenSource{
		base:    &fakeBase{toks: []*oauth2.Token{refreshed}},
		current: initial,
		persist: persist,
	}

	got, err := ts.Token()
	if err != nil || got.AccessToken != "NEW" {
		t.Fatalf("Token() = %+v, %v", got, err)
	}
	if persistCalls != 1 {
		t.Fatalf("persist should fire exactly once on refresh, got %d", persistCalls)
	}

	// The refreshed token landed under the CAPTURED ref/key, and nothing else.
	st, err := openWith(&config.Config{CredentialRef: ref}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	stored, err := st.Token()
	if err != nil || stored.AccessToken != "NEW" {
		t.Fatalf("stored under captured ref = %+v, %v", stored, err)
	}
	keys, err := st.cs.ListBundle(st.profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != KeyOAuthToken {
		t.Fatalf("refresh must create no extra keys; bundle = %v", keys)
	}
	if st.Ref() != ref {
		t.Fatalf("ref changed: %q", st.Ref())
	}

	// Idempotent: no access-token change → no second persist.
	ts.base = &fakeBase{toks: []*oauth2.Token{refreshed}}
	if _, err := ts.Token(); err != nil {
		t.Fatal(err)
	}
	if persistCalls != 1 {
		t.Fatalf("no refresh ⇒ no extra persist, got %d", persistCalls)
	}
}

func TestPersistentTokenSource_PersistFailureIsNonFatal(t *testing.T) {
	ts := &PersistentTokenSource{
		base:    &fakeBase{toks: []*oauth2.Token{{AccessToken: "NEW"}}},
		current: &oauth2.Token{AccessToken: "OLD"},
		persist: func(*oauth2.Token) error { return errors.New("keyring down") },
	}
	got, err := ts.Token()
	if err != nil {
		t.Fatalf("persist failure must be non-fatal, got %v", err)
	}
	if got.AccessToken != "NEW" {
		t.Fatalf("token must still be returned: %+v", got)
	}
}
