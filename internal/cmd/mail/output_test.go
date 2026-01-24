package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessagePrintOptions(t *testing.T) {
	t.Run("default options are all false", func(t *testing.T) {
		opts := MessagePrintOptions{}
		assert.False(t, opts.IncludeThreadID)
		assert.False(t, opts.IncludeTo)
		assert.False(t, opts.IncludeSnippet)
		assert.False(t, opts.IncludeBody)
	})

	t.Run("options can be set individually", func(t *testing.T) {
		opts := MessagePrintOptions{
			IncludeThreadID: true,
			IncludeBody:     true,
		}
		assert.True(t, opts.IncludeThreadID)
		assert.False(t, opts.IncludeTo)
		assert.False(t, opts.IncludeSnippet)
		assert.True(t, opts.IncludeBody)
	})
}
