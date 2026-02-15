package contacts

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestContactsCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "contacts")
	})

	t.Run("has ppl alias", func(t *testing.T) {
		testutil.SliceContains(t, cmd.Aliases, "ppl")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Long)
		testutil.Contains(t, cmd.Long, "read-only")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		testutil.GreaterOrEqual(t, len(subcommands), 4)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		testutil.SliceContains(t, names, "list")
		testutil.SliceContains(t, names, "search")
		testutil.SliceContains(t, names, "get")
		testutil.SliceContains(t, names, "groups")
	})
}

func TestListCommand(t *testing.T) {
	cmd := newListCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "list")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)
	})

	t.Run("rejects arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "m")
		testutil.Equal(t, flag.DefValue, "10")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})
}

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "search <query>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"query"})
		testutil.NoError(t, err)
	})

	t.Run("rejects no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)
	})

	t.Run("rejects multiple arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"query1", "query2"})
		testutil.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "m")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})
}

func TestGetCommand(t *testing.T) {
	cmd := newGetCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "get <resource-name>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"people/c123"})
		testutil.NoError(t, err)
	})

	t.Run("rejects no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})
}

func TestGroupsCommand(t *testing.T) {
	cmd := newGroupsCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "groups")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)
	})

	t.Run("rejects arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "m")
		testutil.Equal(t, flag.DefValue, "30")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})
}
