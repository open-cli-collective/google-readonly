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

## Design Principles

- **Gmail browser parity**: Mail features must integrate with how people expect Gmail to behave in the browser. `gro` is one client among many on a shared mailbox — drafts, quoting, threading, and labels should look and behave like native Gmail when later opened or sent from the web UI. Emit the conventions/markers Gmail's own client recognizes (e.g. wrap reply quotes in `gmail_quote` markup so Gmail collapses them behind its `…` toggle; `Re:` subject handling; RFC threading headers) rather than reimplementing client-side rendering. Prefer Gmail-native parity over technically-valid-but-foreign output.

## Testing

Run tests: `make test`

Coverage: `make test-cover && open coverage.html`

Tests use `internal/testutil/` for assertions (`testutil.Equal`, `testutil.NoError`, etc.) and fixtures (`testutil.SampleMessage()`, `testutil.SampleEvent()`, etc.). See `docs/golden-principles.md` for mock and test helper patterns.

## OAuth2 Configuration

Per the Open CLI Collective Secret-Handling Standard §2.3:

- **OAuth client JSON** (from Google Cloud Console) is *deployment material*,
  not a secret: a plain file `~/.config/google-readonly/oauth_client.json`
  (override with `oauth_client_path` in `config.yml`). The legacy
  `credentials.json` is auto-migrated to it on first run.
- **OAuth token** (the per-user access secret) lives **only** in the OS
  keyring via `cli-common/credstore` — macOS Keychain, Linux Secret Service,
  Windows Credential Manager, or an opt-in encrypted file
  (`keyring.backend: file` + `GOOGLE_READONLY_KEYRING_PASSPHRASE`). No
  `security`/`secret-tool` shell-out, no `token.json` fallback. A legacy
  `token.json` (or old `security`/`secret-tool` item) is migrated into the
  keyring once (§1.8), then removed; a legacy-vs-keyring conflict fails loud.
- **Non-secret config**: OS-native config dir via `cli-common/statedir`
  (`~/Library/Application Support/google-readonly/config.yml` on macOS,
  `%APPDATA%\google-readonly\config.yml` on Windows, `~/.config/
  google-readonly/config.yml` on Linux). Fields: `credential_ref`,
  `oauth_client_path`, `granted_scopes`. A legacy `config.json` and the
  pre-MON-5371 `cache_ttl_hours` field are read transparently once (TTL
  is now hard-coded per resource per working-with-state.md §4.4; the
  field is ignored on load).
- Ingress is only `gro init` (browser flow; `--auth-code-stdin` for two-phase
  installs) or `gro set-credential --key oauth_token --stdin|--from-env`.

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
- `github.com/open-cli-collective/cli-common` - shared `credstore` (OS keyring)
- `gopkg.in/yaml.v3` - `config.yml`

## Common Issues

**"unable to read OAuth client JSON"**: Run `gro init` and follow the OAuth setup wizard (it writes `oauth_client.json`).

**"Token has been expired or revoked"**: Run `gro config clear && gro init`.
