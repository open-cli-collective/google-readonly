package mail

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestReadCommand(t *testing.T) {
	cmd := newReadCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "read <message-id>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)

		err = cmd.Args(cmd, []string{"msg123"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"msg1", "msg2"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
		testutil.Contains(t, cmd.Short, "message")
	})

	t.Run("long description mentions message ID source", func(t *testing.T) {
		testutil.Contains(t, cmd.Long, "search")
	})
}
