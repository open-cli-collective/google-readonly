package config

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestConfigCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "config")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		testutil.GreaterOrEqual(t, len(subcommands), 3)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		testutil.SliceContains(t, names, "show")
		testutil.SliceContains(t, names, "test")
		testutil.SliceContains(t, names, "clear")
	})
}

func TestConfigShowCommand(t *testing.T) {
	cmd := newShowCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "show")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Long)
	})
}

func TestConfigTestCommand(t *testing.T) {
	cmd := newTestCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "test")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Long)
	})
}

func TestConfigClearCommand(t *testing.T) {
	cmd := newClearCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "clear")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Long)
		testutil.Contains(t, cmd.Long, "token")
	})
}
