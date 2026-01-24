package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalendarCommand(t *testing.T) {
	cmd := NewCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "calendar", cmd.Use)
	})

	t.Run("has cal alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "cal")
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Calendar")
	})

	t.Run("has long description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "events")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 5)

		var names []string
		for _, sub := range subcommands {
			names = append(names, sub.Name())
		}
		assert.Contains(t, names, "list")
		assert.Contains(t, names, "events")
		assert.Contains(t, names, "get")
		assert.Contains(t, names, "today")
		assert.Contains(t, names, "week")
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

		err = cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "calendar")
	})
}

func TestEventsCommand(t *testing.T) {
	cmd := newEventsCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "events [calendar-id]", cmd.Use)
	})

	t.Run("accepts optional calendar id argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"calendar-id"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"calendar-id", "extra"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		assert.NotNil(t, flag)
		assert.Equal(t, "m", flag.Shorthand)
		assert.Equal(t, "10", flag.DefValue)
	})

	t.Run("has from flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("from")
		assert.NotNil(t, flag)
	})

	t.Run("has to flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("to")
		assert.NotNil(t, flag)
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
		assert.Equal(t, "c", flag.Shorthand)
		assert.Equal(t, "primary", flag.DefValue)
	})
}

func TestGetCommand(t *testing.T) {
	cmd := newGetCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "get <event-id>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"event-id"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"event-id", "extra"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
		assert.Equal(t, "c", flag.Shorthand)
		assert.Equal(t, "primary", flag.DefValue)
	})
}

func TestTodayCommand(t *testing.T) {
	cmd := newTodayCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "today", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "today")
	})
}

func TestWeekCommand(t *testing.T) {
	cmd := newWeekCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "week", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "week")
	})
}
