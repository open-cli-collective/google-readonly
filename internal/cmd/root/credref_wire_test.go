package root

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

// TestWireCredentialRefSelection_FlagSet proves a --ref on a real command
// path is recorded in the override the keychain.open resolver reads.
func TestWireCredentialRefSelection_FlagSet(t *testing.T) {
	resetState(t)
	t.Setenv(keychain.CredentialRefEnvVar(), "")

	probe := newProbeCmd("probe-ref-flagset")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-ref-flagset", "--ref", "google-readonly/acct-a"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	v, set := keychain.GetCredentialRefOverride()
	if !set {
		t.Fatalf("override flagSet = false, want true")
	}
	if v != "google-readonly/acct-a" {
		t.Errorf("override value = %q, want %q", v, "google-readonly/acct-a")
	}
}

// TestWireCredentialRefSelection_FlagInvalid asserts a malformed --ref fails
// up front with a clear "--ref" error, before any keyring work.
func TestWireCredentialRefSelection_FlagInvalid(t *testing.T) {
	resetState(t)

	probe := newProbeCmd("probe-ref-invalid")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-ref-invalid", "--ref", "no-slash"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--"+credentialRefFlagName) {
		t.Errorf("error should mention --%s: %v", credentialRefFlagName, err)
	}
}

// TestWireCredentialRefSelection_ShadowingSubcommand regresses the
// cobra-doesn't-chain-PersistentPreRunE bug for --ref, mirroring the
// --backend guard.
func TestWireCredentialRefSelection_ShadowingSubcommand(t *testing.T) {
	resetState(t)
	t.Setenv(keychain.CredentialRefEnvVar(), "")

	shadow := &cobra.Command{
		Use: "shadow-ref",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireCredentialRefSelection(cmd)
		},
	}
	leaf := newProbeCmd("leaf")
	shadow.AddCommand(leaf)
	rootCmd.AddCommand(shadow)
	defer removeChild(t, shadow)

	rootCmd.SetArgs([]string{"shadow-ref", "leaf", "--ref", "google-readonly/acct-b"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute through shadowing PreRunE: %v", err)
	}
	v, set := keychain.GetCredentialRefOverride()
	if !set || v != "google-readonly/acct-b" {
		t.Errorf("override = (%q, %v); want (\"google-readonly/acct-b\", true) — shadower's PreRunE failed to invoke WireCredentialRefSelection", v, set)
	}
}

// TestCredentialRef_SetCredentialShadowsPersistent documents the intentional
// exception to the inherit-everywhere rule: `set-credential` keeps its own
// local --ref (the write target), so it must resolve to a DIFFERENT *pflag.Flag
// than the root's persistent selector — while a read command inherits the
// canonical persistent one. A regression that dropped set-credential's local
// flag (or that made a read command shadow --ref) would flip these.
func TestCredentialRef_SetCredentialShadowsPersistent(t *testing.T) {
	canonical := rootCmd.PersistentFlags().Lookup(credentialRefFlagName)
	if canonical == nil {
		t.Fatalf("root persistent flag --%s not registered", credentialRefFlagName)
	}

	var sc *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "set-credential" {
			sc = c
			break
		}
	}
	if sc == nil {
		t.Fatal("set-credential command not registered on rootCmd")
	}
	if got := sc.Flag(credentialRefFlagName); got == nil {
		t.Fatalf("set-credential has no --%s", credentialRefFlagName)
	} else if got == canonical {
		t.Errorf("set-credential --%s resolved to the persistent flag; expected its own local shadow", credentialRefFlagName)
	}

	// A read command (no local --ref) must inherit the canonical persistent flag.
	me := newProbeCmd("probe-ref-inherit")
	rootCmd.AddCommand(me)
	defer removeChild(t, me)
	if got := me.Flag(credentialRefFlagName); got != canonical {
		t.Errorf("read command --%s = %p, want canonical %p (unexpected shadow)", credentialRefFlagName, got, canonical)
	}
}
