package contacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContactsCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "contacts", cmd.Use)
	})

	t.Run("has ppl alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ppl")
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "read-only")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 4)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		assert.Contains(t, names, "list")
		assert.Contains(t, names, "search")
		assert.Contains(t, names, "get")
		assert.Contains(t, names, "groups")
	})
}

func TestListCommand(t *testing.T) {
	cmd := newListCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("rejects arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
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
}

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "search <query>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"query"})
		assert.NoError(t, err)
	})

	t.Run("rejects no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)
	})

	t.Run("rejects multiple arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"query1", "query2"})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		assert.NotNil(t, flag)
		assert.Equal(t, "m", flag.Shorthand)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})
}

func TestGetCommand(t *testing.T) {
	cmd := newGetCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "get <resource-name>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"people/c123"})
		assert.NoError(t, err)
	})

	t.Run("rejects no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})
}

func TestGroupsCommand(t *testing.T) {
	cmd := newGroupsCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "groups", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("rejects arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		assert.NotNil(t, flag)
		assert.Equal(t, "m", flag.Shorthand)
		assert.Equal(t, "30", flag.DefValue)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})
}
