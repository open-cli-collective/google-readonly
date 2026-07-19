# Adding a New Google API Domain

This checklist covers adding a new Google API (e.g., Google Tasks, Google
Sheets) to gro. Because gro is now a thin CLI on
[`google-cli-common`](https://github.com/open-cli-collective/google-cli-common),
adding a domain spans two modules: the reusable **API client** goes in
google-cli-common (so grw and any future sibling can use it too), while the
**command surface**, **scope registration**, and **structural-test wiring** stay
here. Structural tests in `internal/architecture/architecture_test.go`
automatically enforce steps marked [enforced].

## Checklist

### 1. Add the OAuth scope (this repo)

In `internal/appidentity/appidentity.go`, add the readonly scope to `Scopes`
(and a human description to `ScopeDescriptions`):

```go
var Scopes = []string{
    gmail.GmailModifyScope,
    // ...
    tasks.TasksReadonlyScope, // new
}
```

[enforced] `TestAllScopesAreNonDestructive` requires every scope in
`appidentity.Scopes` to be on the non-destructive allowlist. gro must never
request a send/delete/settings scope.

### 2. Create the API client package (google-cli-common)

In the google-cli-common repo, create `{domain}/` with:
- `client.go` — `Client` struct, `NewClient(ctx context.Context) (*Client, error)`, methods
- Data model files as needed
- `*_test.go` — Unit tests for parsing and data models

The constructor follows the established pattern (auth lives in common):

```go
func NewClient(ctx context.Context) (*Client, error) {
    client, err := auth.GetHTTPClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("loading OAuth client: %w", err)
    }
    srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
    if err != nil {
        return nil, fmt.Errorf("creating Tasks service: %w", err)
    }
    return &Client{service: srv}, nil
}
```

Release a new google-cli-common version once the client lands, and bump the
`require` in gro's `go.mod` to it.

### 3. Create the command package (this repo)

Create `internal/cmd/{domain}/` with these files:

**`output.go`** — [enforced] Must contain:
- An exported interface ending in `Client` (e.g., `TasksClient`)
- A `ClientFactory` variable whose default calls the google-cli-common client:
  ```go
  var ClientFactory = func(ctx context.Context) (TasksClient, error) {
      return tasks.NewClient(ctx) // tasks = google-cli-common/tasks
  }
  ```
- Domain-specific text-rendering helpers

> Do **not** add a package-local `printJSON()`. Per golden principle §4 (#144),
> resource-surface leaves emit text only.

**`{domain}.go`** — [enforced] An exported `NewCommand()` returning
`*cobra.Command`, with `AddCommand()` calls for all subcommands.

**Each subcommand file** — [enforced] Resource-surface leaves emit text only;
they must NOT declare `--json/-j` (`TestResourceLeavesHaveNoJSONFlag`).

**`main_test.go`** — a `TestMain` that registers gro's identity so config/
keychain paths resolve in tests:

```go
func TestMain(m *testing.M) {
    config.Register(appidentity.Identity())
    os.Exit(m.Run())
}
```

### 4. Create test infrastructure

**`mock_test.go`** — Function-field mock with a compile-time interface check.
**`handlers_test.go`** — `withMockClient` / `withFailingClientFactory` using the
centralized `testutil.WithFactory` (from google-cli-common). Capture output with
`testutil.CaptureStdout`.

### 5. Add test fixtures (google-cli-common)

In google-cli-common's `testutil/fixtures.go`, add `SampleX()` functions for the
new API types.

### 6. Register the domain command (this repo)

In `internal/cmd/root/root.go`, add:

```go
rootCmd.AddCommand(tasks.NewCommand())
```

### 7. Update structural test registration (this repo)

In `internal/architecture/architecture_test.go`, add the new domain to the
`domainPackages` slice and the `domainCommands()` map.

### 8. Verify

Run `make check`. The structural tests catch any missing patterns. If a new
google-cli-common client was added, make sure `go.mod` requires the version that
contains it.
