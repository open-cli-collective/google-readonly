package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "search <query>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"query"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"query1", "query2"})
		assert.Error(t, err)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		assert.NotNil(t, flag)
		assert.Equal(t, "m", flag.Shorthand)
		assert.Equal(t, "10", flag.DefValue)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has examples in long description", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "from:")
		assert.Contains(t, cmd.Long, "subject:")
		assert.Contains(t, cmd.Long, "is:unread")
	})
}
