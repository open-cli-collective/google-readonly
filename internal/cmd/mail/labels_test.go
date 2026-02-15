package mail

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestLabelsCommand(t *testing.T) {
	cmd := newLabelsCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "labels")
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
		testutil.Contains(t, cmd.Short, "label")
	})
}

func TestGetLabelType(t *testing.T) {
	t.Run("returns category for CATEGORY_ prefix", func(t *testing.T) {
		label := &gmailapi.Label{Id: "CATEGORY_UPDATES", Type: "system"}
		testutil.Equal(t, getLabelType(label), "category")
	})

	t.Run("returns category for all category types", func(t *testing.T) {
		categories := []string{"CATEGORY_SOCIAL", "CATEGORY_PROMOTIONS", "CATEGORY_FORUMS", "CATEGORY_PERSONAL"}
		for _, id := range categories {
			label := &gmailapi.Label{Id: id, Type: "system"}
			testutil.Equal(t, getLabelType(label), "category")
		}
	})

	t.Run("returns system for system type", func(t *testing.T) {
		label := &gmailapi.Label{Id: "INBOX", Type: "system"}
		testutil.Equal(t, getLabelType(label), "system")
	})

	t.Run("returns user for user type", func(t *testing.T) {
		label := &gmailapi.Label{Id: "Label_123", Type: "user"}
		testutil.Equal(t, getLabelType(label), "user")
	})

	t.Run("returns user for empty type", func(t *testing.T) {
		label := &gmailapi.Label{Id: "Label_456", Type: ""}
		testutil.Equal(t, getLabelType(label), "user")
	})
}

func TestLabelTypePriority(t *testing.T) {
	t.Run("user has highest priority (lowest value)", func(t *testing.T) {
		testutil.Equal(t, labelTypePriority("user"), 0)
	})

	t.Run("category is second priority", func(t *testing.T) {
		testutil.Equal(t, labelTypePriority("category"), 1)
	})

	t.Run("system is third priority", func(t *testing.T) {
		testutil.Equal(t, labelTypePriority("system"), 2)
	})

	t.Run("unknown types have lowest priority", func(t *testing.T) {
		testutil.Equal(t, labelTypePriority("unknown"), 3)
		testutil.Equal(t, labelTypePriority(""), 3)
	})

	t.Run("priorities maintain correct sort order", func(t *testing.T) {
		testutil.Less(t, labelTypePriority("user"), labelTypePriority("category"))
		testutil.Less(t, labelTypePriority("category"), labelTypePriority("system"))
		testutil.Less(t, labelTypePriority("system"), labelTypePriority("unknown"))
	})
}

// Tests for truncate moved to internal/format/format_test.go
