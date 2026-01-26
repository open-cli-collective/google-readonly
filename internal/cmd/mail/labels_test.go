package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestLabelsCommand(t *testing.T) {
	cmd := newLabelsCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "labels", cmd.Use)
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
		assert.Contains(t, cmd.Short, "label")
	})
}

func TestGetLabelType(t *testing.T) {
	t.Run("returns category for CATEGORY_ prefix", func(t *testing.T) {
		label := &gmailapi.Label{Id: "CATEGORY_UPDATES", Type: "system"}
		assert.Equal(t, "category", getLabelType(label))
	})

	t.Run("returns category for all category types", func(t *testing.T) {
		categories := []string{"CATEGORY_SOCIAL", "CATEGORY_PROMOTIONS", "CATEGORY_FORUMS", "CATEGORY_PERSONAL"}
		for _, id := range categories {
			label := &gmailapi.Label{Id: id, Type: "system"}
			assert.Equal(t, "category", getLabelType(label), "expected category for %s", id)
		}
	})

	t.Run("returns system for system type", func(t *testing.T) {
		label := &gmailapi.Label{Id: "INBOX", Type: "system"}
		assert.Equal(t, "system", getLabelType(label))
	})

	t.Run("returns user for user type", func(t *testing.T) {
		label := &gmailapi.Label{Id: "Label_123", Type: "user"}
		assert.Equal(t, "user", getLabelType(label))
	})

	t.Run("returns user for empty type", func(t *testing.T) {
		label := &gmailapi.Label{Id: "Label_456", Type: ""}
		assert.Equal(t, "user", getLabelType(label))
	})
}

func TestLabelTypePriority(t *testing.T) {
	t.Run("user has highest priority (lowest value)", func(t *testing.T) {
		assert.Equal(t, 0, labelTypePriority("user"))
	})

	t.Run("category is second priority", func(t *testing.T) {
		assert.Equal(t, 1, labelTypePriority("category"))
	})

	t.Run("system is third priority", func(t *testing.T) {
		assert.Equal(t, 2, labelTypePriority("system"))
	})

	t.Run("unknown types have lowest priority", func(t *testing.T) {
		assert.Equal(t, 3, labelTypePriority("unknown"))
		assert.Equal(t, 3, labelTypePriority(""))
	})

	t.Run("priorities maintain correct sort order", func(t *testing.T) {
		assert.Less(t, labelTypePriority("user"), labelTypePriority("category"))
		assert.Less(t, labelTypePriority("category"), labelTypePriority("system"))
		assert.Less(t, labelTypePriority("system"), labelTypePriority("unknown"))
	})
}

// Tests for truncate moved to internal/format/format_test.go
