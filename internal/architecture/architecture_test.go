package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	mailcmd "github.com/open-cli-collective/google-cli-common/mailcmd"

	"github.com/open-cli-collective/google-readonly/internal/appidentity"
	calcmd "github.com/open-cli-collective/google-readonly/internal/cmd/calendar"
	contactscmd "github.com/open-cli-collective/google-readonly/internal/cmd/contacts"
	drivecmd "github.com/open-cli-collective/google-readonly/internal/cmd/drive"
	mecmd "github.com/open-cli-collective/google-readonly/internal/cmd/me"
)

// domainPackages lists the command packages that must follow structural conventions.
var domainPackages = []string{"calendar", "contacts", "drive", "me"}

// domainCommands returns the top-level cobra.Command for each domain package.
func domainCommands() map[string]*cobra.Command {
	return map[string]*cobra.Command{
		"mail":     mailcmd.NewCommand(),
		"calendar": calcmd.NewCommand(),
		"contacts": contactscmd.NewCommand(),
		"drive":    drivecmd.NewCommand(),
		"me":       mecmd.NewCommand(),
	}
}

// findModuleRoot walks up from the working directory to locate go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (go.mod)")
		}
		dir = parent
	}
}

// parseNonTestFiles parses all non-test .go files in a directory.
func parseNonTestFiles(t *testing.T, dir string) []*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading directory %s: %v", dir, err)
	}
	var files []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		files = append(files, f)
	}
	return files
}

type leafInfo struct {
	path string
	cmd  *cobra.Command
}

// leafCommands recursively collects all leaf commands (commands with no subcommands).
func leafCommands(cmd *cobra.Command, parentPath string) []leafInfo {
	subs := cmd.Commands()
	if len(subs) == 0 {
		return []leafInfo{{path: parentPath, cmd: cmd}}
	}
	var leaves []leafInfo
	for _, sub := range subs {
		subPath := parentPath + " " + sub.Name()
		leaves = append(leaves, leafCommands(sub, subPath)...)
	}
	return leaves
}

// ---------------------------------------------------------------------------
// Structural tests
// ---------------------------------------------------------------------------

// TestDomainPackagesDefineClientInterface verifies that every domain command package
// declares an exported interface type whose name ends in "Client".
func TestDomainPackagesDefineClientInterface(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)

	for _, pkg := range domainPackages {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, "internal", "cmd", pkg)
			files := parseNonTestFiles(t, dir)

			var found bool
			for _, f := range files {
				for _, decl := range f.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok || genDecl.Tok != token.TYPE {
						continue
					}
					for _, spec := range genDecl.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						_, isInterface := typeSpec.Type.(*ast.InterfaceType)
						if isInterface && strings.HasSuffix(typeSpec.Name.Name, "Client") {
							found = true
							if !typeSpec.Name.IsExported() {
								t.Errorf("client interface %s must be exported", typeSpec.Name.Name)
							}
						}
					}
				}
			}

			if !found {
				t.Errorf("package internal/cmd/%s must define an exported interface ending in 'Client' (see docs/golden-principles.md)", pkg)
			}
		})
	}
}

// TestDomainPackagesHaveClientFactory verifies that every domain command package
// declares a package-level ClientFactory variable for dependency injection.
func TestDomainPackagesHaveClientFactory(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)

	for _, pkg := range domainPackages {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, "internal", "cmd", pkg)
			files := parseNonTestFiles(t, dir)

			var found bool
			for _, f := range files {
				for _, decl := range f.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok || genDecl.Tok != token.VAR {
						continue
					}
					for _, spec := range genDecl.Specs {
						valueSpec, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, name := range valueSpec.Names {
							if name.Name == "ClientFactory" {
								found = true
							}
						}
					}
				}
			}

			if !found {
				t.Errorf("package internal/cmd/%s must define a ClientFactory variable for dependency injection (see docs/golden-principles.md)", pkg)
			}
		})
	}
}

// TestDomainPackagesExportNewCommand verifies that every domain command package
// exports a NewCommand() function (top-level, not a method).
func TestDomainPackagesExportNewCommand(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)

	for _, pkg := range domainPackages {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, "internal", "cmd", pkg)
			files := parseNonTestFiles(t, dir)

			var found bool
			for _, f := range files {
				for _, decl := range f.Decls {
					funcDecl, ok := decl.(*ast.FuncDecl)
					if !ok {
						continue
					}
					// Must be a package-level function (no receiver), named NewCommand
					if funcDecl.Name.Name == "NewCommand" && funcDecl.Recv == nil {
						found = true
					}
				}
			}

			if !found {
				t.Errorf("package internal/cmd/%s must export a NewCommand() function (see docs/golden-principles.md)", pkg)
			}
		})
	}
}

// TestResourceLeavesHaveNoJSONFlag verifies the §2 closed-set policy
// from cli-common/docs/output-and-rendering.md: resource-surface leaf
// commands (every leaf under mail/calendar/contacts/drive/me) emit text
// output only. JSON is reserved for control-plane envelopes — today
// that's `gro refresh --json` and `gro config show --json`, neither of
// which is in domainCommands() so neither is touched by this walk.
// Inverted from the pre-#144 TestAllLeafCommandsHaveJSONFlag invariant.
func TestResourceLeavesHaveNoJSONFlag(t *testing.T) {
	t.Parallel()

	for name, cmd := range domainCommands() {
		for _, leaf := range leafCommands(cmd, name) {
			t.Run(strings.TrimSpace(leaf.path), func(t *testing.T) {
				t.Parallel()
				key := strings.TrimSpace(leaf.path)
				if flag := leaf.cmd.Flags().Lookup("json"); flag != nil {
					t.Errorf("resource-surface leaf %q must NOT declare --json (see docs/golden-principles.md §4 + cli-common output-and-rendering §2)", key)
				}
			})
		}
	}
}

// TestResourceLeaf_RejectsJSON_EndToEnd is a spot-check complement to the
// structural walk in TestResourceLeavesHaveNoJSONFlag. It dispatches one
// representative resource leaf with --json through cobra and asserts the
// user-visible "unknown flag" error so the end-to-end contract is exercised,
// not just the static flag set. Closing the closed-set bypass (a new domain
// added outside domainPackages) still requires updating that list — neither
// test compensates for that.
func TestResourceLeaf_RejectsJSON_EndToEnd(t *testing.T) {
	t.Parallel()
	cmd := drivecmd.NewCommand()
	cmd.SetArgs([]string{"drives", "--json"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("gro drive drives --json should error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("expected 'unknown flag' error, got: %v", err)
	}
}

// Dependency-direction invariants (API clients never import cmd; auth never
// imports API clients) are now enforced structurally by the module boundary:
// the gmail/calendar/contacts/drive/people clients and the auth package live in
// the shared google-cli-common module, which has no cmd packages and cannot
// import this main module's internal packages. See google-cli-common for its
// own structural tests.

// allowedScopes is the set of OAuth scopes permitted in appidentity.Scopes.
// Read-only scopes are always safe. Non-readonly scopes are allowed only when
// they enable non-destructive organizational operations (label, archive, star, etc.)
// without granting send or delete access.
var allowedScopes = map[string]bool{
	"https://www.googleapis.com/auth/gmail.readonly":    true,
	"https://www.googleapis.com/auth/gmail.modify":      true, // label, archive, star, read/unread (NOT send/delete)
	"https://www.googleapis.com/auth/calendar.readonly": true,
	"https://www.googleapis.com/auth/calendar.events":   true, // RSVP, color (NOT calendar settings)
	"https://www.googleapis.com/auth/contacts.readonly": true,
	"https://www.googleapis.com/auth/contacts":          true, // group membership, starring (NOT create/delete contacts)
	"https://www.googleapis.com/auth/userinfo.profile":  true, // read authenticated user's name/email for people/me (NOT contacts list)
	"https://www.googleapis.com/auth/drive.readonly":    true,
	"https://www.googleapis.com/auth/drive.metadata":    true, // star/unstar files (NOT file content write)
}

// TestAllScopesAreNonDestructive verifies that every OAuth scope in
// appidentity.Scopes is in the allowlist of non-destructive scopes. This is the
// structural guarantee that keeps gro non-destructive: the scope set it
// registers can never include gmail.settings.* or https://mail.google.com/
// (which would permit filters or permanent deletion).
func TestAllScopesAreNonDestructive(t *testing.T) {
	t.Parallel()

	if len(appidentity.Scopes) == 0 {
		t.Fatal("appidentity.Scopes must not be empty")
	}

	for _, scope := range appidentity.Scopes {
		if !allowedScopes[scope] {
			t.Errorf("scope %q is not in the non-destructive allowlist; update allowedScopes if this scope is safe", scope)
		}
	}
}

// TestNoDestructiveAPIMethodsInProductionCode scans all non-test Go source files
// for Google API destructive method calls. Non-destructive modify methods like
// BatchModify (used for labeling/archiving) are permitted.
func TestNoDestructiveAPIMethodsInProductionCode(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)

	// These patterns are specific to Google API client libraries and unlikely
	// to appear in other contexts. Generic method names like .Delete() or
	// .Insert() are intentionally excluded to avoid false positives.
	// Note: .BatchModify( is intentionally allowed — it's used for bulk label operations.
	forbiddenPatterns := []string{
		".Send(",
		".Trash(",
		".Untrash(",
		".BatchDelete(",
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == ".git" || name == "dist" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("reading %s: %v", path, readErr)
			return nil
		}
		content := string(data)
		rel, _ := filepath.Rel(root, path)

		for _, pattern := range forbiddenPatterns {
			if strings.Contains(content, pattern) {
				t.Errorf("file %s contains forbidden destructive API method %q — this CLI only allows non-destructive operations", rel, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking source tree: %v", err)
	}
}

// commonClientPackages are the google-cli-common API client packages gro relies
// on. gro's non-destructive guarantee depends on these staying non-destructive
// even though they now live in a separate module (the destructive Gmail surface
// belongs to grw, never to the shared clients).
var commonClientPackages = []string{"gmail", "calendar", "contacts", "drive", "people"}

// TestSharedGoogleClientsAreNonDestructive extends the non-destructive guarantee
// across the module boundary. TestNoDestructiveAPIMethodsInProductionCode only
// walks this repo, but the Google API clients gro drives now live in the pinned
// google-cli-common module. This test resolves that module in the local module
// cache and scans its client packages for the same forbidden destructive
// methods, so a future common release that introduces one (e.g. via a shared
// batch helper) fails gro's CI the moment gro bumps to it — instead of silently
// shipping. Keeps the guarantee code-enforced, not prose-only.
func TestSharedGoogleClientsAreNonDestructive(t *testing.T) {
	t.Parallel()

	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}",
		"github.com/open-cli-collective/google-cli-common").Output()
	if err != nil {
		t.Fatalf("resolving google-cli-common module dir: %v", err)
	}
	commonDir := strings.TrimSpace(string(out))
	if commonDir == "" {
		t.Fatal("empty google-cli-common module dir")
	}

	// Same forbidden set as the in-repo scan. .BatchModify( stays allowed
	// (bulk labeling/archiving).
	forbiddenPatterns := []string{".Send(", ".Trash(", ".Untrash(", ".BatchDelete("}

	for _, pkg := range commonClientPackages {
		dir := filepath.Join(commonDir, pkg)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("reading shared client package %s: %v", dir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			data, readErr := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // path from resolved module cache
			if readErr != nil {
				t.Errorf("reading %s/%s: %v", pkg, name, readErr)
				continue
			}
			content := string(data)
			for _, pattern := range forbiddenPatterns {
				if strings.Contains(content, pattern) {
					t.Errorf("shared client google-cli-common/%s/%s contains forbidden destructive API method %q — the clients gro depends on must stay non-destructive (the destructive surface belongs to grw)", pkg, name, pattern)
				}
			}
		}
	}
}
