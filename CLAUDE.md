# CLAUDE.md

This file provides guidance for AI agents working with the gro codebase.

## Project Overview

gro is a **non-destructive** command-line interface for Google services written in Go. It uses OAuth2 for authentication and supports read-only access plus non-destructive organizational operations (labeling, archiving, starring, marking read/unread). No send, delete, or destructive operations are possible.

**Binary name:** `gro`
**Module:** `github.com/open-cli-collective/google-readonly`

### Features
- Gmail: Search, read, thread viewing, labels, attachments, archive, star, mark read/unread, label, categorize, draft (compose-only, never sent — supports reply-to-thread)
- Google Calendar: List calendars, view events, today/week shortcuts
- Google Contacts: List contacts, search, view details, list groups
- Google Drive: List files, search, get details, download, tree view, shared drives

## Quick Commands

```bash
make build          # Build binary
make test           # Run tests with race detection
make test-cover     # Tests with HTML coverage report
make lint           # Run golangci-lint
make fmt            # Format code
make check          # CI gate: tidy, lint, test, build
make install        # Install to /usr/local/bin
```

## Documentation

| Document | Contents |
|----------|----------|
| `docs/architecture.md` | Dependency graph, package responsibilities, file naming conventions |
| `docs/golden-principles.md` | Mechanical rules enforced by structural tests |
| `docs/adding-a-domain.md` | Step-by-step checklist for adding a new Google API |

## Key Constraints

- **Non-destructive by design**: Only allowlisted scopes in `auth.AllScopes`. No destructive API methods (send, delete, trash). Non-destructive modify operations (label, archive, star) are permitted.
- **Interface-at-consumer**: Each `internal/cmd/{domain}/output.go` defines its client interface.
- **ClientFactory DI**: Swappable factory for test mock injection.
- **--json on all leaf commands**: Every leaf subcommand supports `--json/-j`.
- **Structural enforcement**: `internal/architecture/architecture_test.go` enforces all patterns at CI time.

See `docs/golden-principles.md` for the full set of enforced rules.

## Testing

Run tests: `make test`

Coverage: `make test-cover && open coverage.html`

Tests use `internal/testutil/` for assertions (`testutil.Equal`, `testutil.NoError`, etc.) and fixtures (`testutil.SampleMessage()`, `testutil.SampleEvent()`, etc.). See `docs/golden-principles.md` for mock and test helper patterns.

## OAuth2 Configuration

Credentials: `~/.config/google-readonly/credentials.json` (from Google Cloud Console)

Tokens stored securely per platform:
- **macOS**: System Keychain (via `security` CLI)
- **Linux**: libsecret (via `secret-tool`) if available, otherwise config file
- **Fallback**: `~/.config/google-readonly/token.json` with 0600 permissions

## Error Conventions

Follow [Go conventions](https://github.com/go/wiki/wiki/CodeReviewComments#error-strings): lowercase, no trailing punctuation, use `%w` for wrapping.

## Commit Conventions

Use conventional commits: `type(scope): description`

| Prefix | Purpose | Triggers Release? |
|--------|---------|-------------------|
| `feat:` | New features | Yes |
| `fix:` | Bug fixes | Yes |
| `docs:` | Documentation only | No |
| `test:` | Adding/updating tests | No |
| `refactor:` | Code changes (no bug fix or feature) | No |
| `chore:` | Maintenance tasks | No |
| `ci:` | CI/CD changes | No |

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `golang.org/x/oauth2` - OAuth2 client
- `google.golang.org/api/*` - Google API clients (Gmail, Calendar, People, Drive)

## Common Issues

**"Unable to read credentials file"**: Run `gro init` and follow the OAuth setup wizard.

**"Token has been expired or revoked"**: Run `gro config clear && gro init`.
