# Golden Principles

These are the mechanical rules that keep the codebase consistent. Each rule is enforced by structural tests in `internal/architecture/architecture_test.go` and runs automatically in CI via `make check`.

## 1. Interface-at-consumer

Every domain command package (`internal/cmd/{domain}/`) defines its own client interface in `output.go`. The API client (in `google-cli-common`) does NOT define an interface — it returns a concrete `*Client` struct.

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

## 4. Text-only resource leaves (no per-command `--json`)

Per cli-common `docs/output-and-rendering.md` §2, resource-surface leaf commands (every leaf under `mail`, `calendar`, `contacts`, `drive`, `me`) emit text output only. JSON is reserved for control-plane envelopes — today that's `gro refresh --json` (§4.6) and `gro config show --json` (diagnostic). Inverted from the pre-#144 "every leaf must have `--json`" rule.

The `mail` leaves come from the shared `google-cli-common/mailcmd` package (composed into gro's root); `calendar`, `contacts`, `drive`, and `me` are gro's own `internal/cmd/{domain}` packages. All obey the text-only rule.

**Control-plane carve-out criteria.** A command qualifies as a carve-out only if it (a) is not a resource leaf under `mail`/`calendar`/`contacts`/`drive`/`me`, AND (b) emits a control-plane envelope (write confirmation, cache freshness) or diagnostic introspection of CLI state — not a Google API resource. New JSON surfaces should be argued against these criteria before being added.

**Enforced by:** `TestResourceLeavesHaveNoJSONFlag`

## 5. Non-destructive only

All OAuth scopes in `appidentity.Scopes` must appear in the non-destructive allowlist. No destructive API methods (`.Send()`, `.Trash()`, `.BatchDelete()`, etc.) in gro's production code. Non-destructive modify methods like `.BatchModify()` (used for labeling/archiving) are permitted. The shared `google-cli-common/gmail` client is likewise non-destructive; the destructive Gmail surface lives only in grw (google-readwrite).

**Enforced by:** `TestAllScopesAreNonDestructive`, `TestNoDestructiveAPIMethodsInProductionCode`

## 6. Dependency direction

- API client packages must NOT import command packages (clients don't know about commands)
- `auth` must NOT import API client packages (auth is lower-level)

These invariants are now guaranteed by the module boundary: the API clients and `auth` live in `google-cli-common`, which has no command packages and cannot import this module's `internal/`. See google-cli-common for any structural tests over its own tree.

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

These live in `google-cli-common/testutil` and are shared by gro and grw:

- `testutil.CaptureStdout(t, func())` — captures stdout during command execution
- `testutil.WithFactory(&factory, replacement, func())` — generic factory swap
- `testutil.SampleX()` functions — fixture data for all API types
- `testutil.Equal`, `testutil.NoError`, etc. — assertion helpers

gro's own domain command tests register its identity via a `TestMain` that calls `config.Register(appidentity.Identity())`, so DirName-derived config/keychain paths resolve.
