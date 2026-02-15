package root

import (
	"testing"

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
		testutil.Contains(t, rootCmd.Long, "read-only")
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
	})
}
