package drive

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "search [query]")
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"query"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"query1", "query2"})
		testutil.Error(t, err)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "m")
		testutil.Equal(t, flag.DefValue, "25")
	})

	t.Run("has name flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("name")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "n")
	})

	t.Run("has type flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("type")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "t")
	})

	t.Run("has owner flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("owner")
		testutil.NotNil(t, flag)
	})

	t.Run("has modified-after flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("modified-after")
		testutil.NotNil(t, flag)
	})

	t.Run("has modified-before flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("modified-before")
		testutil.NotNil(t, flag)
	})

	t.Run("has in-folder flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("in-folder")
		testutil.NotNil(t, flag)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.Contains(t, cmd.Short, "Search")
	})
}

func TestBuildSearchQuery(t *testing.T) {
	t.Run("builds full-text search query", func(t *testing.T) {
		query, err := buildSearchQuery("quarterly report", false, "", "", "", "", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "trashed = false")
		testutil.Contains(t, query, "fullText contains 'quarterly report'")
	})

	t.Run("builds name-only search query", func(t *testing.T) {
		query, err := buildSearchQuery("budget", true, "", "", "", "", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "name contains 'budget'")
		testutil.NotContains(t, query, "fullText")
	})

	t.Run("adds type filter", func(t *testing.T) {
		query, err := buildSearchQuery("test", false, "document", "", "", "", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType = 'application/vnd.google-apps.document'")
	})

	t.Run("returns error for invalid type", func(t *testing.T) {
		_, err := buildSearchQuery("test", false, "invalid", "", "", "", "")
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "unknown file type")
	})

	t.Run("adds owner filter with 'me'", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "me", "", "", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "'me' in owners")
	})

	t.Run("adds owner filter with email", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "john@example.com", "", "", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "'john@example.com' in owners")
	})

	t.Run("adds modified-after filter", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "", "2024-01-01", "", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "modifiedTime > '2024-01-01T00:00:00'")
	})

	t.Run("adds modified-before filter", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "", "", "2024-12-31", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "modifiedTime < '2024-12-31T23:59:59'")
	})

	t.Run("adds folder scope", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "", "", "", "", "folder123")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "'folder123' in parents")
	})

	t.Run("combines multiple filters", func(t *testing.T) {
		query, err := buildSearchQuery("report", false, "document", "me", "2024-01-01", "", "folder123")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "trashed = false")
		testutil.Contains(t, query, "fullText contains 'report'")
		testutil.Contains(t, query, "mimeType = 'application/vnd.google-apps.document'")
		testutil.Contains(t, query, "'me' in owners")
		testutil.Contains(t, query, "modifiedTime > '2024-01-01T00:00:00'")
		testutil.Contains(t, query, "'folder123' in parents")
	})

	t.Run("builds query with no search term", func(t *testing.T) {
		query, err := buildSearchQuery("", false, "document", "", "", "", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "trashed = false")
		testutil.Contains(t, query, "mimeType")
		testutil.NotContains(t, query, "fullText")
		testutil.NotContains(t, query, "name contains")
	})
}

func TestEscapeQueryString(t *testing.T) {
	t.Run("escapes single quotes", func(t *testing.T) {
		result := escapeQueryString("it's a test")
		testutil.Equal(t, result, "it\\'s a test")
	})

	t.Run("handles string without quotes", func(t *testing.T) {
		result := escapeQueryString("simple query")
		testutil.Equal(t, result, "simple query")
	})

	t.Run("handles multiple quotes", func(t *testing.T) {
		result := escapeQueryString("don't won't can't")
		testutil.Equal(t, result, "don\\'t won\\'t can\\'t")
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := escapeQueryString("")
		testutil.Equal(t, result, "")
	})
}
