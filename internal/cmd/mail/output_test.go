package mail

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestMessagePrintOptions(t *testing.T) {
	t.Run("default options are all false", func(t *testing.T) {
		opts := MessagePrintOptions{}
		testutil.False(t, opts.IncludeThreadID)
		testutil.False(t, opts.IncludeTo)
		testutil.False(t, opts.IncludeSnippet)
		testutil.False(t, opts.IncludeBody)
	})

	t.Run("options can be set individually", func(t *testing.T) {
		opts := MessagePrintOptions{
			IncludeThreadID: true,
			IncludeBody:     true,
		}
		testutil.True(t, opts.IncludeThreadID)
		testutil.False(t, opts.IncludeTo)
		testutil.False(t, opts.IncludeSnippet)
		testutil.True(t, opts.IncludeBody)
	})
}
