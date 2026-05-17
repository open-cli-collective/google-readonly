// Package migrationsink is the leaf holding the §1.8 one-time-migration
// signal. It has no internal dependencies so both internal/keychain (the
// recorder) and internal/output (the JSON splicer) can import it without an
// import cycle — gro has no global JSON state, and the migration fires deep
// inside keychain.Open(), so the signal is recorded here run-scoped and
// delivered at output time (JSON splice) or flushed to stderr by
// root.runRoot. Consume-once, so it appears exactly once and never leaks
// into a later response or a parallel test.
package migrationsink

import (
	"encoding/json"
	"io"
	"sync"
)

var (
	mu      sync.Mutex
	pending []byte // marshaled migration block value, or nil if none
	human   string // human stderr line for the non-JSON path
)

// Record stores the §1.8 block (anything that marshals to the `_migration`
// value, e.g. credstore.MigrationBlock) plus the human line for the stderr
// path. Never calling this means "no migration this run".
func Record(block interface{}, humanLine string) {
	b, err := json.Marshal(block)
	if err != nil {
		return // a marshal failure here must not break the actual command
	}
	mu.Lock()
	pending = b
	human = humanLine
	mu.Unlock()
}

// Take returns the pending block + human line and clears both
// (consume-once). The block is nil when nothing is pending.
func Take() (block []byte, humanLine string) {
	mu.Lock()
	defer mu.Unlock()
	b, h := pending, human
	pending, human = nil, ""
	return b, h
}

// Reset drops any pending record without emitting it. Test hook so one
// test's recorded migration can never bleed into another.
func Reset() {
	mu.Lock()
	pending, human = nil, ""
	mu.Unlock()
}

// FlushMigrationNotice emits the §1.8 human line to w iff a record is still
// pending (no JSON path consumed it), then consumes it. Wired once as a
// deferred call in root.runRoot so it fires on success AND error, above the
// os.Exit trap. Writing to stderr never corrupts a --json stdout body.
func FlushMigrationNotice(w io.Writer) {
	_, h := Take()
	if h != "" {
		_, _ = io.WriteString(w, h+"\n")
	}
}
