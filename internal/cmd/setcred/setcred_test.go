package setcred

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/credtest"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

const tokenJSON = `{"access_token":"SECRET-ACCESS","refresh_token":"SECRET-REFRESH","token_type":"Bearer"}`

func TestSetCredentialStdin(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: keychain.KeyOAuthToken, stdin: true, in: strings.NewReader(tokenJSON)})
	if err != nil {
		t.Fatalf("set-credential --stdin: %v", err)
	}
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	tok, err := st.Token()
	if err != nil || tok.AccessToken != "SECRET-ACCESS" {
		t.Fatalf("token not stored: %+v err=%v", tok, err)
	}
}

func TestSetCredentialKeyAllowlist(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: "not_allowed", stdin: true, in: strings.NewReader(tokenJSON)})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("want allowlist error, got %v", err)
	}
}

func TestSetCredentialFromEnvEmptyNamesVar(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: keychain.KeyOAuthToken, fromEnv: "GRO_TEST_UNSET_VAR"})
	if err == nil || !strings.Contains(err.Error(), "GRO_TEST_UNSET_VAR") {
		t.Fatalf("error must name the empty env var, got %v", err)
	}
}

func TestSetCredentialRejectsNonToken(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: keychain.KeyOAuthToken, stdin: true, in: strings.NewReader(`{"not":"a token"}`)})
	if err == nil || !strings.Contains(err.Error(), "neither an access nor a refresh token") {
		t.Fatalf("want token-shape rejection, got %v", err)
	}
}
