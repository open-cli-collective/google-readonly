package root

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

const serviceName = "google-readonly"

// resetState resets the package-level flag override. Child-command
// cleanup is the responsibility of each test (via removeChild) because
// the children added during a test are local to it; resetState has no
// reference to them and cannot strip them generically. Tests that add
// children MUST `defer removeChild(t, child)` immediately after
// AddCommand to keep package-level rootCmd clean for subsequent tests
// (notably TestWireBackendSelection_RealCommandTreeInheritsFlag, which
// walks the entire tree).
func resetState(t *testing.T) {
	t.Helper()
	keychain.SetBackendFlagOverride("", false)
	// rootCmd.SetArgs mutates package-level state; if a test panics before
	// the next test calls SetArgs, a stale slice could bleed in (notably
	// under `go test -shuffle=on`). Clear it on cleanup.
	t.Cleanup(func() {
		keychain.SetBackendFlagOverride("", false)
		rootCmd.SetArgs(nil)
	})
}

// newProbeCmd returns a no-op subcommand used to exercise the root's
// PersistentPreRunE through a real Execute() call.
func newProbeCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:  name,
		RunE: func(*cobra.Command, []string) error { return nil },
	}
}

// removeChild detaches a child added during a test so package-level
// rootCmd stays clean for the next test.
func removeChild(t *testing.T, child *cobra.Command) {
	t.Helper()
	rootCmd.RemoveCommand(child)
}

func TestWireBackendSelection_FlagSet(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")

	probe := newProbeCmd("probe-flagset")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-flagset", "--backend", "memory"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	v, set := keychain.GetBackendFlagOverride()
	if !set {
		t.Fatalf("override flagSet = false, want true")
	}
	if v != "memory" {
		t.Errorf("override value = %q, want %q", v, "memory")
	}
}

func TestWireBackendSelection_FlagInvalid(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")

	probe := newProbeCmd("probe-flaginvalid")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-flaginvalid", "--backend", "bogus"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, cccredstore.ErrBackendNotImplemented) {
		t.Errorf("errors.Is(_, ErrBackendNotImplemented) = false; err=%v", err)
	}
	if !strings.Contains(err.Error(), "backend") {
		t.Errorf("error should mention --backend: %v", err)
	}
}

// TestWireBackendSelection_ConfigPassthrough exercises BindBackendFlag
// directly: this is the contract our wiring depends on at the openWith
// layer (where cfg.Keyring.Backend is bound).
func TestWireBackendSelection_ConfigPassthrough(t *testing.T) {
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")
	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, "", false, "memory"); err != nil {
		t.Fatalf("BindBackendFlag: %v", err)
	}
	if opts.Backend != "" {
		t.Errorf("Backend = %q, want empty (no flag)", opts.Backend)
	}
	if opts.ConfigBackend != cccredstore.BackendMemory {
		t.Errorf("ConfigBackend = %q, want %q", opts.ConfigBackend, cccredstore.BackendMemory)
	}
}

// TestWireBackendSelection_InvalidConfigDeferred asserts BindBackendFlag
// does NOT validate the config-side value; the failure surfaces later at
// credstore.Open.
func TestWireBackendSelection_InvalidConfigDeferred(t *testing.T) {
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")
	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, "", false, "bogus"); err != nil {
		t.Fatalf("BindBackendFlag should NOT validate config: %v", err)
	}
	if string(opts.ConfigBackend) != "bogus" {
		t.Errorf("ConfigBackend = %q, want verbatim passthrough %q", opts.ConfigBackend, "bogus")
	}
}

// TestWireBackendSelection_ShadowingSubcommand regresses the
// cobra-doesn't-chain-PersistentPreRunE bug. gro has no shadowers
// today, but the exported WireBackendSelection helper exists for
// exactly this case.
func TestWireBackendSelection_ShadowingSubcommand(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")

	shadow := &cobra.Command{
		Use: "shadow",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireBackendSelection(cmd)
		},
	}
	leaf := newProbeCmd("leaf")
	shadow.AddCommand(leaf)
	rootCmd.AddCommand(shadow)
	defer removeChild(t, shadow)

	rootCmd.SetArgs([]string{"shadow", "leaf", "--backend", "memory"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute through shadowing PreRunE: %v", err)
	}
	v, set := keychain.GetBackendFlagOverride()
	if !set || v != "memory" {
		t.Errorf("override = (%q, %v); want (\"memory\", true) — shadower's PreRunE failed to invoke WireBackendSelection", v, set)
	}
}

// TestWireBackendSelection_RealCommandTreeInheritsFlag walks the real
// gro command tree (everything init() registers) and asserts every leaf
// resolves --backend to the same *pflag.Flag pointer as the root's
// PersistentFlags entry. Pointer-identity catches a leaf that locally
// shadows --backend with its own string flag (pflag would return the
// shadow, not the inherited persistent flag).
func TestWireBackendSelection_RealCommandTreeInheritsFlag(t *testing.T) {
	canonical := rootCmd.PersistentFlags().Lookup(cccredstore.BackendFlagName)
	if canonical == nil {
		t.Fatalf("root persistent flag --%s not registered", cccredstore.BackendFlagName)
	}

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		children := c.Commands()
		if len(children) == 0 {
			got := c.Flag(cccredstore.BackendFlagName)
			if got == nil {
				t.Errorf("%q: cmd.Flag(--%s) returned nil", c.CommandPath(), cccredstore.BackendFlagName)
				return
			}
			if got != canonical {
				t.Errorf("%q: cmd.Flag(--%s) pointer = %p, want canonical %p (local shadowing?)",
					c.CommandPath(), cccredstore.BackendFlagName, got, canonical)
			}
			return
		}
		for _, child := range children {
			walk(child)
		}
	}
	walk(rootCmd)
}
