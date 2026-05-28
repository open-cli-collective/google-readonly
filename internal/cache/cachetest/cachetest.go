// Package cachetest exposes test seams for internal/cache. It is named
// with the `test` suffix so production code in any sibling package may
// not import it without raising eyebrows; importing this package outside
// a *_test.go file is a code-review red flag.
package cachetest

import (
	"time"

	"github.com/open-cli-collective/google-readonly/internal/cache"
)

// SwapClock swaps the cache package's clock for a test, returning a restore
// function that the caller should defer. Intended for cross-package tests
// that need to advance time past a cache TTL.
func SwapClock(fn func() time.Time) (restore func()) {
	return cache.SwapClockForTest(fn)
}
