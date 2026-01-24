package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadCommand(t *testing.T) {
	cmd := newReadCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "read <message-id>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"msg123"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"msg1", "msg2"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "message")
	})

	t.Run("long description mentions message ID source", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "search")
	})
}
