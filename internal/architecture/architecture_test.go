package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/auth"
	calcmd "github.com/open-cli-collective/google-readonly/internal/cmd/calendar"
	contactscmd "github.com/open-cli-collective/google-readonly/internal/cmd/contacts"
	drivecmd "github.com/open-cli-collective/google-readonly/internal/cmd/drive"
	mailcmd "github.com/open-cli-collective/google-readonly/internal/cmd/mail"
)

// domainPackages lists the command packages that must follow structural conventions.
var domainPackages = []string{"mail", "calendar", "contacts", "drive"}

// apiClientPackages lists the internal API client package directory names.
var apiClientPackages = []string{"gmail", "calendar", "contacts", "drive"}

// jsonExemptCommands lists leaf commands exempt from the --json flag requirement.
// Key format: "parent subcommand" (e.g., "mail attachments download").
// Only add exemptions for commands that output binary file data where JSON is inapplicable.
var jsonExemptCommands = map[string]bool{
	"mail attachments download": true, // writes binary attachment files to disk
	"drive download":            true, // writes binary file data to disk
}

// domainCommands returns the top-level cobra.Command for each domain package.
func domainCommands() map[string]*cobra.Command {
	return map[string]*cobra.Command{
		"mail":     mailcmd.NewCommand(),
		"calendar": calcmd.NewCommand(),
		"contacts": contactscmd.NewCommand(),
		"drive":    drivecmd.NewCommand(),
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

// collectImports returns all import paths from a set of parsed files.
func collectImports(files []*ast.File) []string {
	var imports []string
	for _, f := range files {
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			imports = append(imports, path)
		}
	}
	return imports
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

// TestAllLeafCommandsHaveJSONFlag verifies that every leaf subcommand
// (commands with no children) declares a --json/-j flag.
func TestAllLeafCommandsHaveJSONFlag(t *testing.T) {
	t.Parallel()

	for name, cmd := range domainCommands() {
		for _, leaf := range leafCommands(cmd, name) {
			t.Run(strings.TrimSpace(leaf.path), func(t *testing.T) {
				t.Parallel()
				key := strings.TrimSpace(leaf.path)
				if jsonExemptCommands[key] {
					t.Skipf("exempt from --json requirement")
				}
				flag := leaf.cmd.Flags().Lookup("json")
				if flag == nil {
					t.Errorf("leaf command %q must have a --json flag (see docs/golden-principles.md)", key)
					return
				}
				if flag.Shorthand != "j" {
					t.Errorf("leaf command %q --json flag must have shorthand 'j', got %q", key, flag.Shorthand)
				}
			})
		}
	}
}

// TestAPIClientPackagesDoNotImportCmd verifies that API client packages
// (internal/gmail, internal/calendar, etc.) never import command packages.
// Dependency direction must be: cmd -> api client, never the reverse.
func TestAPIClientPackagesDoNotImportCmd(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)

	for _, pkg := range apiClientPackages {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(root, "internal", pkg)
			files := parseNonTestFiles(t, dir)
			imports := collectImports(files)

			for _, imp := range imports {
				if strings.Contains(imp, "internal/cmd") {
					t.Errorf("API client package internal/%s must not import cmd packages, but imports %q", pkg, imp)
				}
			}
		})
	}
}

// TestAuthPackageDoesNotImportAPIClients verifies that the auth package
// does not depend on any internal API client packages.
// Dependency direction must be: api client -> auth, never the reverse.
func TestAuthPackageDoesNotImportAPIClients(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)

	dir := filepath.Join(root, "internal", "auth")
	files := parseNonTestFiles(t, dir)
	imports := collectImports(files)

	for _, imp := range imports {
		for _, apiPkg := range apiClientPackages {
			if strings.HasSuffix(imp, "/internal/"+apiPkg) {
				t.Errorf("auth package must not import API client package internal/%s", apiPkg)
			}
		}
	}
}

// TestAllScopesAreReadOnly verifies that every OAuth scope in auth.AllScopes
// is a read-only scope. This is the primary mechanical enforcement of the
// read-only-by-design principle.
func TestAllScopesAreReadOnly(t *testing.T) {
	t.Parallel()

	if len(auth.AllScopes) == 0 {
		t.Fatal("auth.AllScopes must not be empty")
	}

	for _, scope := range auth.AllScopes {
		if !strings.Contains(scope, "readonly") {
			t.Errorf("scope %q is not a readonly scope; all scopes in AllScopes must be read-only", scope)
		}
	}
}

// TestNoWriteAPIMethodsInProductionCode scans all non-test Go source files
// for Google API write method calls. This is defense-in-depth on top of the
// scope check — even with readonly scopes, we don't want write method calls
// in the codebase since they indicate incorrect intent.
func TestNoWriteAPIMethodsInProductionCode(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)

	// These patterns are specific to Google API client libraries and unlikely
	// to appear in other contexts. Generic method names like .Delete() or
	// .Insert() are intentionally excluded to avoid false positives.
	forbiddenPatterns := []string{
		".Send(",
		".Trash(",
		".Untrash(",
		".BatchModify(",
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
				t.Errorf("file %s contains forbidden write API method %q — this CLI is read-only by design", rel, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking source tree: %v", err)
	}
}
