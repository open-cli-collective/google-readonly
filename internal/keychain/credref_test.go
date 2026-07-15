package keychain

import "testing"

// resetCredRefOverride keeps the package-level --ref override clean across
// tests so a leaked value can't tilt the next.
func resetCredRefOverride(t *testing.T) {
	t.Helper()
	SetCredentialRefOverride("", false)
	t.Cleanup(func() { SetCredentialRefOverride("", false) })
}

// TestCredentialRefEnvVar pins the derived env-var name to the documented
// <SERVICE>_CREDENTIAL_REF, tracking the same prefix as the backend env var.
func TestCredentialRefEnvVar(t *testing.T) {
	if got := CredentialRefEnvVar(); got != "GOOGLE_READONLY_CREDENTIAL_REF" {
		t.Errorf("CredentialRefEnvVar() = %q, want %q", got, "GOOGLE_READONLY_CREDENTIAL_REF")
	}
}

// TestEffectiveRef_Precedence proves --ref flag > env > config, and that an
// override is reported (so the caller suppresses the one-time migration).
func TestEffectiveRef_Precedence(t *testing.T) {
	const cfgRef = "google-readonly/cfg"

	t.Run("config only", func(t *testing.T) {
		resetCredRefOverride(t)
		t.Setenv(CredentialRefEnvVar(), "")
		ref, ov := effectiveRef(cfgRef)
		if ref != cfgRef || ov {
			t.Errorf("got (%q,%v), want (%q,false)", ref, ov, cfgRef)
		}
	})

	t.Run("env overrides config", func(t *testing.T) {
		resetCredRefOverride(t)
		t.Setenv(CredentialRefEnvVar(), "google-readonly/env")
		ref, ov := effectiveRef(cfgRef)
		if ref != "google-readonly/env" || !ov {
			t.Errorf("got (%q,%v), want (google-readonly/env,true)", ref, ov)
		}
	})

	t.Run("flag overrides env and config", func(t *testing.T) {
		resetCredRefOverride(t)
		t.Setenv(CredentialRefEnvVar(), "google-readonly/env")
		SetCredentialRefOverride("google-readonly/flag", true)
		ref, ov := effectiveRef(cfgRef)
		if ref != "google-readonly/flag" || !ov {
			t.Errorf("got (%q,%v), want (google-readonly/flag,true)", ref, ov)
		}
	})

	t.Run("flag set but empty does not override", func(t *testing.T) {
		resetCredRefOverride(t)
		t.Setenv(CredentialRefEnvVar(), "")
		SetCredentialRefOverride("", true) // Changed=true but no value
		ref, ov := effectiveRef(cfgRef)
		if ref != cfgRef || ov {
			t.Errorf("got (%q,%v), want (%q,false) — empty --ref must fall through", ref, ov, cfgRef)
		}
	})
}
