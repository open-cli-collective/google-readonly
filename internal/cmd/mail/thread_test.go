package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThreadCommand(t *testing.T) {
	cmd := newThreadCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "thread <id>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"thread123"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"thread1", "thread2"})
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
		assert.Contains(t, cmd.Short, "thread")
	})

	t.Run("long description explains thread ID", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "thread ID")
		assert.Contains(t, cmd.Long, "message ID")
	})
}
