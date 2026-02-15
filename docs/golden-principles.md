# Golden Principles

These are the mechanical rules that keep the codebase consistent. Each rule is enforced by structural tests in `internal/architecture/architecture_test.go` and runs automatically in CI via `make check`.

## 1. Interface-at-consumer

Every domain command package (`internal/cmd/{domain}/`) defines its own client interface in `output.go`. The API client package (`internal/{domain}/`) does NOT define an interface — it returns a concrete `*Client` struct.

**Enforced by:** `TestDomainPackagesDefineClientInterface`

## 2. ClientFactory for dependency injection

Every domain command package declares a package-level `ClientFactory` variable. Production code calls `ClientFactory(ctx)`. Tests override it to inject mocks.

```go
var ClientFactory = func(ctx context.Context) (XClient, error) {
    return x.NewClient(ctx)
}
```

**Enforced by:** `TestDomainPackagesHaveClientFactory`

## 3. NewCommand() factory

Parent commands export `NewCommand()` returning `*cobra.Command`. Subcommands use unexported `new{Sub}Command()`. Parent commands register subcommands via `cmd.AddCommand()`.

**Enforced by:** `TestDomainPackagesExportNewCommand`

## 4. --json on every leaf command

All leaf subcommands (commands with no children) support `--json/-j` for machine-readable output. Download commands that output binary file data are exempt.

**Enforced by:** `TestAllLeafCommandsHaveJSONFlag`

## 5. Read-only only

Only `*ReadonlyScope` constants may appear in `auth.AllScopes`. No write API methods (`.Send()`, `.Trash()`, `.BatchModify()`, etc.) in production code.

**Enforced by:** `TestAllScopesAreReadOnly`, `TestNoWriteAPIMethodsInProductionCode`

## 6. Dependency direction

- API client packages must NOT import `internal/cmd/` (clients don't know about commands)
- `internal/auth/` must NOT import API client packages (auth is lower-level)

**Enforced by:** `TestAPIClientPackagesDoNotImportCmd`, `TestAuthPackageDoesNotImportAPIClients`

## 7. context.Context on all I/O methods

Every public method that performs I/O takes `context.Context` as its first parameter. The only exceptions are pure getter methods that return cached data (e.g., `GetLabelName`, `GetLabels`).

## 8. Error wrapping

Use `fmt.Errorf("doing X: %w", err)` at every level. Error messages are lowercase and have no trailing punctuation, following [Go conventions](https://github.com/go/wiki/wiki/CodeReviewComments#error-strings).

## 9. Mock pattern

Mocks use function fields in `mock_test.go` with a compile-time interface check:

```go
type MockXClient struct {
    MethodFunc func(...) (...)
}

var _ XClient = (*MockXClient)(nil)

func (m *MockXClient) Method(...) (...) {
    if m.MethodFunc != nil {
        return m.MethodFunc(...)
    }
    return zero, nil
}
```

Test helpers `withMockClient` and `withFailingClientFactory` use `testutil.WithFactory` to swap the `ClientFactory`.

## 10. Centralized test helpers

- `testutil.CaptureStdout(t, func())` — captures stdout during command execution
- `testutil.WithFactory(&factory, replacement, func())` — generic factory swap
- `testutil.SampleX()` functions — fixture data for all API types
- `testutil.Equal`, `testutil.NoError`, etc. — assertion helpers
