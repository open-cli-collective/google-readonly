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
		testutil.Contains(t, cmd.Long, "organizing")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		testutil.GreaterOrEqual(t, len(subcommands), 8)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		testutil.SliceContains(t, names, "list")
		testutil.SliceContains(t, names, "search")
		testutil.SliceContains(t, names, "get")
		testutil.SliceContains(t, names, "groups")
		testutil.SliceContains(t, names, "add-to-group")
		testutil.SliceContains(t, names, "remove-from-group")
		testutil.SliceContains(t, names, "star")
		testutil.SliceContains(t, names, "unstar")
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

	t.Run("has ids flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("ids")
		testutil.NotNil(t, flag)
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

	t.Run("has ids flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("ids")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.DefValue, "false")
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

func TestAddToGroupCommand(t *testing.T) {
	cmd := newAddToGroupCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Contains(t, cmd.Use, "add-to-group")
	})

	t.Run("requires at least one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"Friends"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"Friends", "people/c123"})
		testutil.NoError(t, err)
	})

	t.Run("rejects no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has dry-run flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("dry-run")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "n")
	})

	t.Run("has stdin flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("stdin")
		testutil.NotNil(t, flag)
	})

	t.Run("has query flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("query")
		testutil.NotNil(t, flag)
	})
}

func TestRemoveFromGroupCommand(t *testing.T) {
	cmd := newRemoveFromGroupCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Contains(t, cmd.Use, "remove-from-group")
	})

	t.Run("requires at least one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"Friends"})
		testutil.NoError(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has dry-run flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("dry-run")
		testutil.NotNil(t, flag)
	})
}

func TestStarCommand(t *testing.T) {
	cmd := newStarCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Contains(t, cmd.Use, "star")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has dry-run flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("dry-run")
		testutil.NotNil(t, flag)
	})

	t.Run("has stdin flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("stdin")
		testutil.NotNil(t, flag)
	})

	t.Run("has query flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("query")
		testutil.NotNil(t, flag)
	})
}

func TestUnstarCommand(t *testing.T) {
	cmd := newUnstarCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Contains(t, cmd.Use, "unstar")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has dry-run flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("dry-run")
		testutil.NotNil(t, flag)
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
