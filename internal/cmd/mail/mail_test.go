package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "mail", cmd.Use)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 5)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		assert.Contains(t, names, "search")
		assert.Contains(t, names, "read")
		assert.Contains(t, names, "thread")
		assert.Contains(t, names, "labels")
		assert.Contains(t, names, "attachments")
	})
}
