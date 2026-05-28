package bulk

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestResult_Print_Text(t *testing.T) {
	r := &Result{Action: "archived", IDs: []string{"a", "b"}, Count: 2}
	out := testutil.CaptureStdout(t, func() {
		err := r.Print()
		testutil.NoError(t, err)
	})
	testutil.Contains(t, out, "Archived 2 message(s).")
}

func TestResult_Print_TextDryRun(t *testing.T) {
	r := &Result{Action: "archive", IDs: []string{"a", "b"}, Count: 2, DryRun: true}
	out := testutil.CaptureStdout(t, func() {
		err := r.Print()
		testutil.NoError(t, err)
	})
	testutil.Contains(t, out, "[dry-run] Would archive 2 message(s).")
}

func TestCapitalize(t *testing.T) {
	t.Parallel()
	testutil.Equal(t, capitalize("archived"), "Archived")
	testutil.Equal(t, capitalize("starred"), "Starred")
	testutil.Equal(t, capitalize(""), "")
}
