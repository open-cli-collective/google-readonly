# Adding a New Google API Domain

This checklist covers adding a new Google API (e.g., Google Tasks, Google Sheets) to gro. The structural tests in `internal/architecture/architecture_test.go` automatically enforce steps marked with [enforced].

## Checklist

### 1. Add the OAuth scope

In `internal/auth/auth.go`, add the readonly scope to `AllScopes`:
```go
var AllScopes = []string{
    gmail.GmailReadonlyScope,
    calendar.CalendarReadonlyScope,
    people.ContactsReadonlyScope,
    drive.DriveReadonlyScope,
    tasks.TasksReadonlyScope, // new
}
```

[enforced] Only `*ReadonlyScope` constants are permitted.

### 2. Create the API client package

Create `internal/{domain}/` with:
- `client.go` — `Client` struct, `NewClient(ctx context.Context) (*Client, error)`, methods
- Data model files as needed
- `*_test.go` — Unit tests for parsing and data models

The constructor must follow the established pattern:
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

[enforced] This package must NOT import any `internal/cmd/` package.

### 3. Create the command package

Create `internal/cmd/{domain}/` with these files:

**`output.go`** — [enforced] Must contain:
- An exported interface ending in `Client` (e.g., `TasksClient`)
- A `ClientFactory` variable
- A `newXClient()` wrapper function
- A `printJSON()` function

**`{domain}.go`** — [enforced] Must contain:
- An exported `NewCommand()` function returning `*cobra.Command`
- `AddCommand()` calls for all subcommands

**Each subcommand file** — [enforced] Each leaf command must have `--json/-j` flag:
- Unexported `new{Sub}Command()` factory
- `--json/-j` flag for JSON output (exempt for binary download commands)

### 4. Create test infrastructure

**`mock_test.go`** — Function-field mock with compile-time interface check:
```go
type MockTasksClient struct {
    ListTasksFunc func(ctx context.Context, ...) (...)
}

var _ TasksClient = (*MockTasksClient)(nil)
```

**`handlers_test.go`** — Test helpers using centralized utilities:
```go
func withMockClient(mock TasksClient, f func()) {
    testutil.WithFactory(&ClientFactory, func(_ context.Context) (TasksClient, error) {
        return mock, nil
    }, f)
}

func withFailingClientFactory(f func()) {
    testutil.WithFactory(&ClientFactory, func(_ context.Context) (TasksClient, error) {
        return nil, errors.New("connection failed")
    }, f)
}
```

Use `testutil.CaptureStdout(t, func() { ... })` for output capture.

### 5. Add test fixtures

In `internal/testutil/fixtures.go`, add `SampleX()` functions for the new API types.

### 6. Register the domain command

In `internal/cmd/root/root.go`, add:
```go
cmd.AddCommand(tasks.NewCommand())
```

### 7. Update structural test registration

In `internal/architecture/architecture_test.go`, add the new domain to:
- `domainPackages` slice (e.g., `"tasks"`)
- `apiClientPackages` slice (e.g., `"tasks"`)
- `domainCommands()` map

### 8. Verify

Run `make check`. The structural tests will catch any missing patterns.
