package root

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "gro", rootCmd.Use)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, rootCmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, rootCmd.Long)
		assert.Contains(t, rootCmd.Long, "read-only")
	})

	t.Run("has version set", func(t *testing.T) {
		assert.NotEmpty(t, rootCmd.Version)
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := rootCmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 3)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		assert.Contains(t, names, "init")
		assert.Contains(t, names, "config")
		assert.Contains(t, names, "mail")
	})
}
