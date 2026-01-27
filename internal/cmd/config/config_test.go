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
		assert.GreaterOrEqual(t, len(subcommands), 4)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		assert.Contains(t, names, "show")
		assert.Contains(t, names, "test")
		assert.Contains(t, names, "clear")
		assert.Contains(t, names, "cache")
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

func TestCacheCommand(t *testing.T) {
	cmd := newCacheCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "cache", cmd.Use)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "cache")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.Equal(t, 3, len(subcommands))

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		assert.Contains(t, names, "show")
		assert.Contains(t, names, "clear")
		assert.Contains(t, names, "ttl")
	})
}

func TestCacheShowCommand(t *testing.T) {
	cmd := newCacheShowCommand()

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
		assert.Contains(t, cmd.Long, "cache")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestCacheClearCommand(t *testing.T) {
	cmd := newCacheClearCommand()

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
	})
}

func TestCacheTTLCommand(t *testing.T) {
	cmd := newCacheTTLCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "ttl <hours>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"24"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"24", "extra"})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "TTL")
	})
}
