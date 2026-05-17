package root

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

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
