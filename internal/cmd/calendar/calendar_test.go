package calendar

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestCalendarCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "calendar")
	})

	t.Run("has cal alias", func(t *testing.T) {
		testutil.SliceContains(t, cmd.Aliases, "cal")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
		testutil.Contains(t, cmd.Short, "Calendar")
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Long)
		testutil.Contains(t, cmd.Long, "events")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		testutil.GreaterOrEqual(t, len(subcommands), 5)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		testutil.SliceContains(t, names, "list")
		testutil.SliceContains(t, names, "events")
		testutil.SliceContains(t, names, "get")
		testutil.SliceContains(t, names, "today")
		testutil.SliceContains(t, names, "week")
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

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
		testutil.Contains(t, cmd.Short, "calendar")
	})
}

func TestEventsCommand(t *testing.T) {
	cmd := newEventsCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "events [calendar-id]")
	})

	t.Run("accepts optional calendar id argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"calendar-id"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"calendar-id", "extra"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "m")
		testutil.Equal(t, flag.DefValue, "10")
	})

	t.Run("has from flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("from")
		testutil.NotNil(t, flag)
	})

	t.Run("has to flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("to")
		testutil.NotNil(t, flag)
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "c")
		testutil.Equal(t, flag.DefValue, "primary")
	})
}

func TestGetCommand(t *testing.T) {
	cmd := newGetCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "get <event-id>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)

		err = cmd.Args(cmd, []string{"event-id"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"event-id", "extra"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "c")
		testutil.Equal(t, flag.DefValue, "primary")
	})
}

func TestTodayCommand(t *testing.T) {
	cmd := newTodayCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "today")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		testutil.NotNil(t, flag)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
		testutil.Contains(t, cmd.Short, "today")
	})
}

func TestWeekCommand(t *testing.T) {
	cmd := newWeekCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "week")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		testutil.NotNil(t, flag)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.NotEmpty(t, cmd.Short)
		testutil.Contains(t, cmd.Short, "week")
	})
}
