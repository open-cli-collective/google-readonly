package mail

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "search <query>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)

		err = cmd.Args(cmd, []string{"query"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"query1", "query2"})
		testutil.Error(t, err)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "m")
		testutil.Equal(t, flag.DefValue, "10")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has examples in long description", func(t *testing.T) {
		testutil.Contains(t, cmd.Long, "from:")
		testutil.Contains(t, cmd.Long, "subject:")
		testutil.Contains(t, cmd.Long, "is:unread")
	})
}
