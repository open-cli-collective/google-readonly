# google-readonly Development Guide

This is the repo-local source for working on `gro`. It contains google-readonly-specific facts only; shared Open CLI standards and automation live in the linked source repositories.

## Repository

Binary: `gro`
Module: `github.com/open-cli-collective/google-readonly`
Entrypoint: `cmd/gro/main.go`
Root command: `internal/cmd/root`

`gro` is a non-destructive command-line interface for Google services. It supports read access plus non-destructive organization operations such as labeling, archiving, starring, marking read or unread, RSVP/color operations, group membership, and drafts that are never sent automatically. No send, delete, trash, or destructive file/contact/event operations should be exposed.

## Repo-Local Sources

### Standards Index

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/STANDARDS.md
Local convenience copy, if present: `STANDARDS.md`

### Golden Principles

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/docs/golden-principles.md
Local convenience copy, if present: `docs/golden-principles.md`

### Architecture

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/docs/architecture.md
Local convenience copy, if present: `docs/architecture.md`

### Adding a Google API Domain

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/docs/adding-a-domain.md
Local convenience copy, if present: `docs/adding-a-domain.md`

### Workspace Admin Setup

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/WORKSPACE_ADMINS.md
Local convenience copy, if present: `WORKSPACE_ADMINS.md`

## Shared Sources

### Shared Open CLI Standards

Source of truth: https://github.com/open-cli-collective/cli-common/tree/main/docs
Local convenience copy, if present: `../cli-common/docs`

Relevant shared docs:

| Document | Use for |
| --- | --- |
| `repo-layout.md` | Repository layout, required files, Makefile target names, lint config, Go version policy, branch settings, commit hygiene |
| `ci.md` | Shared CI behavior and how workflows consume Makefile targets |
| `command-surface.md` | Commands, arguments, flags, prompts, aliases, and mutation-safety conventions |
| `output-and-rendering.md` | Text-first resource output, JSON carve-outs, stream discipline, color, pagination, presenter boundaries |
| `working-with-secrets.md` | Credential ingress, keyring storage, migration behavior, redaction, no-leak testing |
| `working-with-state.md` | Config/cache locations, credential references, cache freshness, state migration, hermetic tests |
| `scriptability.md` | Non-interactive setup, env-bridge flags, health checks, OAuth browser handoff |
| `release.md` and `distribution.md` | Shared release and installation behavior |

### Shared Automation

Source of truth: https://github.com/open-cli-collective/.github
Local convenience copy, if present: `../.github`

## Supported Surfaces

- Gmail: search, read, thread viewing, labels, attachments, archive, star, mark read/unread, label, categorize, and draft compose-only flows, including reply-to-thread.
- Calendar: list calendars, view events, today/week shortcuts, RSVP, and color operations.
- Contacts: list contacts, search, view details, list groups, group membership, and starring.
- Drive: list files, search, get details, download, tree view, shared drives, and starring.

Gmail features should preserve browser parity. `gro` is one client among many on the same mailbox, so drafts, quoting, threading, labels, `Re:` handling, and RFC threading headers should behave naturally when later opened from Gmail.

## Quick Commands

```bash
make build
make test
make test-cover
make lint
make fmt
make check
make install
```

`make build` writes `bin/gro` from `./cmd/gro`. `make test` runs the full test suite with the race detector. `make test-cover` writes `coverage.out` and `coverage.html`. `make check` runs tidy, lint, test, and build.

## Core Constraints

- OAuth scopes live in `auth.AllScopes` and must remain on the non-destructive allowlist enforced by structural tests.
- Production code must not call destructive Google API methods such as send, trash, untrash, or batch delete.
- Each `internal/cmd/{domain}` package defines its own client interface in `output.go`.
- Each domain command package exposes a `ClientFactory` variable for test injection.
- Resource-surface leaf commands under `mail`, `calendar`, `contacts`, `drive`, and `me` emit text only and must not declare `--json` or `-j`.
- JSON is reserved for control-plane or diagnostic envelopes such as `gro refresh --json` and `gro config show --json`.
- `internal/architecture/architecture_test.go` enforces the mechanical architecture rules.

## OAuth, Secrets, and State

OAuth client JSON is deployment material and may live at `~/.config/google-readonly/oauth_client.json`, or at the configured `oauth_client_path`. Legacy `credentials.json` is migrated to `oauth_client.json` on first run.

OAuth tokens are per-user access secrets and live only through `cli-common/credstore`. Backend selection follows the shared precedence rules; `gro`'s env knob is `GOOGLE_READONLY_KEYRING_BACKEND`. The encrypted file backend also requires `GOOGLE_READONLY_KEYRING_PASSPHRASE`.

Non-secret config uses the OS-native config directory via `cli-common/statedir`. Current config fields include `credential_ref`, `oauth_client_path`, `granted_scopes`, and `keyring.backend`.

Secret ingress belongs in setup and credential-management commands only. For the shared rules, use `working-with-secrets.md`, `working-with-state.md`, and `scriptability.md` from `cli-common`.

## Testing Notes

Run repo sanity with `make check`. Structural tests live in `internal/architecture/architecture_test.go`; fixtures and assertions live in `internal/testutil`.

Mock clients use function fields plus compile-time interface checks. Test helpers such as `testutil.WithFactory`, `testutil.CaptureStdout`, `testutil.Equal`, and `testutil.NoError` are the default local patterns.

## Dependencies

- `github.com/spf13/cobra` for the command surface.
- `golang.org/x/oauth2` for OAuth2 client behavior.
- `google.golang.org/api/*` for Gmail, Calendar, People, and Drive APIs.
- `github.com/open-cli-collective/cli-common` for credential storage and state directory behavior.
- `gopkg.in/yaml.v3` for `config.yml`.

## Common Issues

`unable to read OAuth client JSON`: run `gro init` and follow the OAuth setup wizard, or provide the organization-managed OAuth client JSON from `WORKSPACE_ADMINS.md`.

`Token has been expired or revoked`: run `gro config clear` and then `gro init`.
