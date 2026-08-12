package architecture

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// versionPkg is the package whose ldflags -X stamps set gro's version. It
// moved from internal/version to the shared module in the google-cli-common
// refactor; the ldflags in .goreleaser.yaml and the Makefile kept pointing at
// the removed path, and Go silently ignores -X against a nonexistent package
// — so every release from that refactor until this test shipped reporting
// "gro dev (commit: unknown, built: unknown)". If the version package ever
// moves again, this test fails until the ldflags move with it.
const versionPkg = "github.com/open-cli-collective/google-cli-common/version"

// ldflagXRe matches an ldflags -X target's package path (the part before the
// final .Var=value).
var ldflagXRe = regexp.MustCompile(`-X ([\w./-]+)\.(?:Version|Commit|Date)=`)

func TestLdflagsStampTheLinkedVersionPackage(t *testing.T) {
	root := repoRoot(t)
	for _, file := range []string{".goreleaser.yaml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, file)) //nolint:gosec // repo-local test input
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		matches := ldflagXRe.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			t.Fatalf("%s: no -X version ldflags found — version stamping removed?", file)
		}
		for _, m := range matches {
			if m[1] != versionPkg {
				t.Errorf("%s stamps %q, but the linked version package is %q — -X against a package the binary doesn't contain is silently ignored and releases report \"dev\"", file, m[1], versionPkg)
			}
		}
	}
}

// repoRoot walks up from the test's working directory to the go.mod dir.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test dir")
		}
		dir = parent
	}
}
