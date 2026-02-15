# Architecture

## Dependency Graph

```
cmd/gro/main.go
  -> internal/cmd/root/
       -> internal/cmd/mail/       (MailClient interface + ClientFactory)
       -> internal/cmd/calendar/   (CalendarClient interface + ClientFactory)
       -> internal/cmd/contacts/   (ContactsClient interface + ClientFactory)
       -> internal/cmd/drive/      (DriveClient interface + ClientFactory)
       -> internal/cmd/initcmd/    (OAuth setup wizard)
       -> internal/cmd/config/     (Credential management)

Each cmd/ package depends on its API client:
  internal/cmd/mail/     -> internal/gmail/
  internal/cmd/calendar/ -> internal/calendar/
  internal/cmd/contacts/ -> internal/contacts/
  internal/cmd/drive/    -> internal/drive/

All API clients depend on:
  internal/auth/    -> internal/keychain/, internal/config/

Shared utilities (no internal deps):
  internal/testutil/    Test fixtures and assertion helpers
  internal/output/      JSON output encoding
  internal/format/      Human-readable formatting
  internal/errors/      Error types
  internal/log/         Logging
  internal/cache/       Response caching
  internal/zip/         Secure zip extraction
  internal/version/     Build-time version injection
```

## Data Flow

```
User -> cobra command -> ClientFactory(ctx) -> API Client -> auth.GetHTTPClient -> Google API
                                                   |
                                            internal/{gmail,calendar,contacts,drive}/
```

## Package Responsibilities

| Package | Responsibility |
|---------|---------------|
| `cmd/gro/` | Entry point, calls `root.NewCommand()` |
| `internal/cmd/root/` | Root cobra command, registers all domain commands |
| `internal/cmd/{domain}/` | Command handlers, client interface, output formatting |
| `internal/{gmail,calendar,contacts,drive}/` | API client, data models, response parsing |
| `internal/auth/` | OAuth2 config loading, HTTP client creation |
| `internal/keychain/` | Platform-specific secure token storage |
| `internal/testutil/` | Test assertions, fixtures, helpers |
| `internal/architecture/` | Structural tests enforcing codebase conventions |

## File Naming Conventions

Each domain command package (`internal/cmd/{domain}/`) contains:

| File | Purpose |
|------|---------|
| `{domain}.go` | Parent command with `NewCommand()` and `AddCommand()` calls |
| `output.go` | Client interface, `ClientFactory`, `printJSON()`, text formatters |
| `{subcommand}.go` | One file per subcommand with `new{Sub}Command()` factory |
| `mock_test.go` | Mock client with function fields + compile-time interface check |
| `handlers_test.go` | `withMockClient()`, `withFailingClientFactory()`, integration tests |
| `*_test.go` | Additional unit tests |

Each API client package (`internal/{domain}/`) contains:

| File | Purpose |
|------|---------|
| `client.go` | `Client` struct, `NewClient(ctx)`, client methods |
| Additional `.go` | Data models, parsing helpers |
| `*_test.go` | Unit tests |

## Structural Enforcement

Architectural invariants are enforced by tests in `internal/architecture/architecture_test.go`. These run as part of `make check` and CI. See `docs/golden-principles.md` for the rules being enforced.
