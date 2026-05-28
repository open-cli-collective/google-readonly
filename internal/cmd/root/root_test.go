package root

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-readonly/internal/migrationsink"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestRootCommand(t *testing.T) {
	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, rootCmd.Use, "gro")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, rootCmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.NotEmpty(t, rootCmd.Long)
		testutil.Contains(t, rootCmd.Long, "non-destructive")
	})

	t.Run("has version set", func(t *testing.T) {
		testutil.NotEmpty(t, rootCmd.Version)
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := rootCmd.Commands()
		testutil.GreaterOrEqual(t, len(subcommands), 5)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		testutil.SliceContains(t, names, "init")
		testutil.SliceContains(t, names, "config")
		testutil.SliceContains(t, names, "mail")
		testutil.SliceContains(t, names, "calendar")
		testutil.SliceContains(t, names, "contacts")
		testutil.SliceContains(t, names, "set-credential")
	})
}

// TestRunRootFlushesMigrationNoticeOnError proves the real defer placement:
// runRoot's deferred FlushMigrationNotice fires even when the command errors
// (Cobra would skip a PersistentPostRunE here), and before any os.Exit
// (os.Exit lives in ExecuteContext, strictly after runRoot returns).
func TestRunRootFlushesMigrationNoticeOnError(t *testing.T) {
	migrationsink.Reset()
	t.Cleanup(migrationsink.Reset)
	migrationsink.Record(
		credstore.NewMigrationBlock(credstore.MigrationJSONEntry(
			"oauth_token", "file:/x/token.json", "keyring:google-readonly/default/oauth_token")),
		"gro: migrated oauth_token into keyring google-readonly/default")

	// Force ExecuteContext to error without any network/keyring: an unknown
	// subcommand makes cobra return an error (a post-run hook would be
	// skipped here, but runRoot's defer must not be).
	rootCmd.SetArgs([]string{"this-command-does-not-exist"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	r, w, _ := os.Pipe()
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	err := runRoot(context.Background())

	os.Stderr = orig
	_ = w.Close()
	stderr := <-done

	if err == nil {
		t.Fatal("expected runRoot to return the unknown-command error")
	}
	if !strings.Contains(stderr, "migrated oauth_token") {
		t.Fatalf("runRoot's deferred flush must surface the §1.8 notice on error; stderr=%q", stderr)
	}
	var again bytes.Buffer
	migrationsink.FlushMigrationNotice(&again)
	if again.Len() != 0 {
		t.Fatalf("notice must be consume-once, got %q", again.String())
	}
}

func TestNoColorFlagRegistered(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("no-color")
	if f == nil {
		t.Fatal("--no-color persistent flag not registered on rootCmd")
	}
	if f.Value.Type() != "bool" {
		t.Fatalf("expected --no-color to be a bool flag, got %s", f.Value.Type())
	}
}

// withRenderer swaps the lipgloss default renderer for the duration of the
// test, restoring the saved renderer on cleanup. Tests using this must not
// call t.Parallel — the default renderer is process-global.
func withRenderer(t *testing.T, profile termenv.Profile) {
	t.Helper()
	saved := lipgloss.DefaultRenderer()
	t.Cleanup(func() { lipgloss.SetDefaultRenderer(saved) })

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(profile)
	lipgloss.SetDefaultRenderer(r)
}

func TestPersistentPreRunE_NoColorTrueFlipsToAscii(t *testing.T) {
	// Baseline: force ANSI so the flip is detectable.
	withRenderer(t, termenv.ANSI)

	// Reset the package-level flag binding on cleanup so subsequent tests
	// or runs don't see a leaked true.
	t.Cleanup(func() { noColor = false })
	noColor = true

	if err := rootCmd.PersistentPreRunE(rootCmd, nil); err != nil {
		t.Fatalf("PersistentPreRunE returned error: %v", err)
	}

	if got := lipgloss.DefaultRenderer().ColorProfile(); got != termenv.Ascii {
		t.Fatalf("expected ColorProfile == Ascii after noColor=true, got %v", got)
	}
}

// TestNoColorFlagThroughCobra proves the full wiring: cobra parses
// --no-color, sets the bound `noColor` package variable, the existing
// PersistentPreRunE picks it up, and the lipgloss renderer is flipped to
// Ascii. Unlike the direct PreRunE invocation above, this exercises the
// real argv → flag binding → PreRunE chain.
func TestNoColorFlagThroughCobra(t *testing.T) {
	withRenderer(t, termenv.ANSI)

	probe := &cobra.Command{
		Use:  "probe-no-color-flag-wiring",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}
	rootCmd.AddCommand(probe)
	t.Cleanup(func() {
		rootCmd.RemoveCommand(probe)
		rootCmd.SetArgs(nil)
		noColor = false
	})

	rootCmd.SetArgs([]string{"--no-color", "probe-no-color-flag-wiring"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !noColor {
		t.Fatal("expected cobra to set noColor=true after parsing --no-color")
	}
	if got := lipgloss.DefaultRenderer().ColorProfile(); got != termenv.Ascii {
		t.Fatalf("expected Ascii via real flag-binding path, got %v", got)
	}
}

func TestPersistentPreRunE_NoColorFalseLeavesRendererUntouched(t *testing.T) {
	// Baseline: force ANSI.
	withRenderer(t, termenv.ANSI)

	t.Cleanup(func() { noColor = false })
	noColor = false

	// WireBackendSelection may succeed or fail in the test env; we don't
	// care about its return — we care that the renderer wasn't mutated.
	_ = rootCmd.PersistentPreRunE(rootCmd, nil)

	if got := lipgloss.DefaultRenderer().ColorProfile(); got != termenv.ANSI {
		t.Fatalf("expected renderer untouched when noColor=false, got %v", got)
	}
}
