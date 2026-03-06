package bulk

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/google-readonly/internal/output"
)

// Result represents the outcome of a bulk operation.
type Result struct {
	Action   string   `json:"action"`
	IDs      []string `json:"ids"`
	Count    int      `json:"count"`
	DryRun   bool     `json:"dryRun"`
	Details  any      `json:"details,omitempty"`
	ItemNoun string   `json:"-"` // e.g. "message", "file", "contact"; defaults to "message"
}

// Print outputs the result as text or JSON.
func (r *Result) Print(jsonOutput bool) error {
	if jsonOutput {
		return output.JSONStdout(r)
	}
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
