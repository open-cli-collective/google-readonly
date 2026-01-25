package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "search [query]", cmd.Use)
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"query"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"query1", "query2"})
		assert.Error(t, err)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		assert.NotNil(t, flag)
		assert.Equal(t, "m", flag.Shorthand)
		assert.Equal(t, "25", flag.DefValue)
	})

	t.Run("has name flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("name")
		assert.NotNil(t, flag)
		assert.Equal(t, "n", flag.Shorthand)
	})

	t.Run("has type flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("type")
		assert.NotNil(t, flag)
		assert.Equal(t, "t", flag.Shorthand)
	})

	t.Run("has owner flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("owner")
		assert.NotNil(t, flag)
	})

	t.Run("has modified-after flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("modified-after")
		assert.NotNil(t, flag)
	})

	t.Run("has modified-before flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("modified-before")
		assert.NotNil(t, flag)
	})

	t.Run("has in-folder flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("in-folder")
		assert.NotNil(t, flag)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.Contains(t, cmd.Short, "Search")
	})
}

func TestBuildSearchQuery(t *testing.T) {
	t.Run("builds full-text search query", func(t *testing.T) {
		query, err := buildSearchQuery("quarterly report", false, "", "", "", "", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "trashed = false")
		assert.Contains(t, query, "fullText contains 'quarterly report'")
	})

	t.Run("builds name-only search query", func(t *testing.T) {
		query, err := buildSearchQuery("budget", true, "", "", "", "", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "name contains 'budget'")
		assert.NotContains(t, query, "fullText")
	})

	t.Run("adds type filter", func(t *testing.T) {
		query, err := buildSearchQuery("test", false, "document", "", "", "", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType = 'application/vnd.google-apps.document'")
	})

	t.Run("returns error for invalid type", func(t *testing.T) {
		_, err := buildSearchQuery("test", false, "invalid", "", "", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown file type")
	})

	t.Run("adds owner filter with 'me'", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "me", "", "", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "'me' in owners")
	})

	t.Run("adds owner filter with email", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "john@example.com", "", "", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "'john@example.com' in owners")
	})

	t.Run("adds modified-after filter", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "", "2024-01-01", "", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "modifiedTime > '2024-01-01T00:00:00'")
	})

	t.Run("adds modified-before filter", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "", "", "2024-12-31", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "modifiedTime < '2024-12-31T23:59:59'")
	})

	t.Run("adds folder scope", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "", "", "", "folder123")
		assert.NoError(t, err)
		assert.Contains(t, query, "'folder123' in parents")
	})

	t.Run("combines multiple filters", func(t *testing.T) {
		query, err := buildSearchQuery("report", false, "document", "me", "2024-01-01", "", "folder123")
		assert.NoError(t, err)
		assert.Contains(t, query, "trashed = false")
		assert.Contains(t, query, "fullText contains 'report'")
		assert.Contains(t, query, "mimeType = 'application/vnd.google-apps.document'")
		assert.Contains(t, query, "'me' in owners")
		assert.Contains(t, query, "modifiedTime > '2024-01-01T00:00:00'")
		assert.Contains(t, query, "'folder123' in parents")
	})

	t.Run("builds query with no search term", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "document", "", "", "", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "trashed = false")
		assert.Contains(t, query, "mimeType")
		assert.NotContains(t, query, "fullText")
		assert.NotContains(t, query, "name contains")
	})
}

func TestEscapeQueryString(t *testing.T) {
	t.Run("escapes single quotes", func(t *testing.T) {
		result := escapeQueryString("it's a test")
		assert.Equal(t, "it\\'s a test", result)
	})

	t.Run("handles string without quotes", func(t *testing.T) {
		result := escapeQueryString("simple query")
		assert.Equal(t, "simple query", result)
	})

	t.Run("handles multiple quotes", func(t *testing.T) {
		result := escapeQueryString("don't won't can't")
		assert.Equal(t, "don\\'t won\\'t can\\'t", result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := escapeQueryString("")
		assert.Equal(t, "", result)
	})
}
