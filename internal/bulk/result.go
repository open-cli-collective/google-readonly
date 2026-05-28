package bulk

import (
	"fmt"
	"strings"
)

// Result represents the outcome of a bulk operation.
type Result struct {
	Action   string
	IDs      []string
	Count    int
	DryRun   bool
	Details  any
	ItemNoun string // e.g. "message", "file", "contact"; defaults to "message"
}

// Print outputs the result as text.
func (r *Result) Print() error {
	noun := r.ItemNoun
	if noun == "" {
		noun = "message"
	}
	if r.DryRun {
		fmt.Printf("[dry-run] Would %s %d %s(s).\n", r.Action, r.Count, noun)
	} else {
		fmt.Printf("%s %d %s(s).\n", capitalize(r.Action), r.Count, noun)
	}
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
