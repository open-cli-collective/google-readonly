package keychain

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

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

// TestApplyCredentialRefOverride proves the safety-critical part open() relies
// on: a present override swaps cfg.CredentialRef AND forces runMigration=false
// (so the one-time legacy migration never runs against an arbitrary
// --ref/env-selected profile), while no override leaves both untouched. This is
// the open()-side coverage the pure effectiveRef/wiring tests don't provide.
func TestApplyCredentialRefOverride(t *testing.T) {
	const cfgRef = "google-readonly/cfg"

	t.Run("no override: cfg untouched, runMigration passthrough", func(t *testing.T) {
		resetCredRefOverride(t)
		t.Setenv(CredentialRefEnvVar(), "")
		cfg := &config.Config{CredentialRef: cfgRef}
		if rm := applyCredentialRefOverride(cfg, true); !rm {
			t.Errorf("runMigration = %v, want true (passthrough)", rm)
		}
		if cfg.CredentialRef != cfgRef {
			t.Errorf("cfg.CredentialRef = %q, want unchanged %q", cfg.CredentialRef, cfgRef)
		}
		// passthrough must preserve a false caller value too (OpenNoMigrate path)
		if rm := applyCredentialRefOverride(cfg, false); rm {
			t.Errorf("runMigration = %v, want false (passthrough of caller's false)", rm)
		}
	})

	t.Run("flag override: cfg swapped, migration suppressed", func(t *testing.T) {
		resetCredRefOverride(t)
		t.Setenv(CredentialRefEnvVar(), "")
		SetCredentialRefOverride("google-readonly/flag", true)
		cfg := &config.Config{CredentialRef: cfgRef}
		if rm := applyCredentialRefOverride(cfg, true); rm {
			t.Errorf("runMigration = %v, want false (override must suppress migration)", rm)
		}
		if cfg.CredentialRef != "google-readonly/flag" {
			t.Errorf("cfg.CredentialRef = %q, want google-readonly/flag", cfg.CredentialRef)
		}
	})

	t.Run("env override: cfg swapped, migration suppressed", func(t *testing.T) {
		resetCredRefOverride(t)
		t.Setenv(CredentialRefEnvVar(), "google-readonly/env")
		cfg := &config.Config{CredentialRef: cfgRef}
		if rm := applyCredentialRefOverride(cfg, true); rm {
			t.Errorf("runMigration = %v, want false (env override must suppress migration)", rm)
		}
		if cfg.CredentialRef != "google-readonly/env" {
			t.Errorf("cfg.CredentialRef = %q, want google-readonly/env", cfg.CredentialRef)
		}
	})
}
