package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "config", cmd.Use)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 3)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		assert.Contains(t, names, "show")
		assert.Contains(t, names, "test")
		assert.Contains(t, names, "clear")
	})
}

func TestConfigShowCommand(t *testing.T) {
	cmd := newShowCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "show", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
	})
}

func TestConfigTestCommand(t *testing.T) {
	cmd := newTestCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "test", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
	})
}

func TestConfigClearCommand(t *testing.T) {
	cmd := newClearCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "clear", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "token")
	})
}
