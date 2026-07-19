# Architecture

gro is a thin CLI built on the shared
[`google-cli-common`](https://github.com/open-cli-collective/google-cli-common)
module. Almost everything reusable — the Google OAuth flow, the API clients,
credential/config/cache state, rendering and bulk helpers, and the shared cobra
command packages — lives in google-cli-common and is consumed here as a
dependency. This repo contains only what is gro-specific: its identity, its own
domain commands, and its structural tests.

## Dependency Graph

```
cmd/gro/main.go
  -> config.Register(appidentity.Identity())   // stamps DirName, scopes, product name
  -> internal/cmd/root/                         // root command + rootutil global-flag wiring
       # shared command packages (google-cli-common):
       -> mailcmd/       (mail: search/read/archive/label/move/... )
       -> initcmd/       (OAuth setup wizard)
       -> configcmd/     (config management)
       -> setcred/       (set-credential)
       -> refreshcmd/    (cache refresh)
       # gro's own domain command packages (this repo):
       -> internal/cmd/calendar/   (CalendarClient interface + ClientFactory)
       -> internal/cmd/contacts/   (ContactsClient interface + ClientFactory)
       -> internal/cmd/drive/      (DriveClient interface + ClientFactory)
       -> internal/cmd/me/         (PeopleClient interface + ClientFactory)

gro's domain command packages depend on google-cli-common's API clients:
  internal/cmd/calendar/ -> google-cli-common/calendar
  internal/cmd/contacts/ -> google-cli-common/contacts
  internal/cmd/drive/    -> google-cli-common/drive
  internal/cmd/me/       -> google-cli-common/people
  (mailcmd, in common, uses google-cli-common/gmail)

Provided by google-cli-common (imported, not in this repo):
  config, keychain, auth            Config, OS-keyring token storage, OAuth flow
  gmail/calendar/contacts/drive/people   API clients and data models
  bulk, output, format, view, errors, log, cache, zip, version   Helpers
  testutil, credtest, migrationsink  Test fixtures, hermetic creds, migration sink
```

## Data Flow

```
User -> cobra command -> ClientFactory(ctx) -> google-cli-common API client
      -> google-cli-common/auth.GetHTTPClient (keyring token, resolved via the
         registered config.Identity) -> Google API
```

## The identity seam

Everything CLI-specific is funneled through `config.Identity` (defined in
google-cli-common), which `cmd/gro/main.go` registers exactly once at startup
via `config.Register(appidentity.Identity())`, before any config/keychain/auth
call. `DirName` alone drives the config/cache directory, the keyring service
segment, and the derived `<SERVICE>_KEYRING_BACKEND` /
`<SERVICE>_KEYRING_PASSPHRASE` / `<SERVICE>_CREDENTIAL_REF` env vars. gro's
identity — its dir name, default credential ref, product name, and OAuth scope
set — lives in `internal/appidentity/appidentity.go`.

## Package Responsibilities

| Package | Responsibility |
|---------|---------------|
| `cmd/gro/` | Entry point: registers `appidentity.Identity()`, then runs `root.ExecuteContext` |
| `internal/appidentity/` | gro's `config.Identity`: dir name, default ref, product name, OAuth scopes (+ descriptions) |
| `internal/cmd/root/` | Root cobra command; registers the shared command packages and gro's domain commands; global-flag wiring delegated to `google-cli-common/rootutil` |
| `internal/cmd/{calendar,contacts,drive,me}/` | gro's own domain command handlers, client interface, and output formatting |
| `internal/architecture/` | Structural tests enforcing codebase conventions |
| `internal/noleak/` | End-to-end secret no-leak tests |
| `google-cli-common/*` | Everything shared: OAuth/credential/config infra, API clients, rendering/bulk helpers, and the `mailcmd`/`initcmd`/`configcmd`/`setcred`/`refreshcmd`/`rootutil` command packages |

## File Naming Conventions

Each gro-owned domain command package (`internal/cmd/{domain}/`) contains:

| File | Purpose |
|------|---------|
| `{domain}.go` | Parent command with `NewCommand()` and `AddCommand()` calls |
| `output.go` | Client interface, `ClientFactory`, text formatters (no `printJSON()` — see #144) |
| `{subcommand}.go` | One file per subcommand with `new{Sub}Command()` factory |
| `mock_test.go` | Mock client with function fields + compile-time interface check |
| `handlers_test.go` | `withMockClient()`, `withFailingClientFactory()`, integration tests |
| `main_test.go` | `TestMain` registering `appidentity.Identity()` so config/keychain paths resolve |
| `*_test.go` | Additional unit tests |

The API client packages (`gmail`, `calendar`, `contacts`, `drive`, `people`)
now live in google-cli-common; see that repo for their `client.go` /
`NewClient(ctx)` conventions.

## Structural Enforcement

Architectural invariants are enforced by tests in
`internal/architecture/architecture_test.go`. These run as part of `make check`
and CI. See `docs/golden-principles.md` for the rules being enforced. Note that
the dependency-direction invariants (API clients never import cmd; auth never
imports API clients) are now guaranteed by the module boundary — those packages
live in google-cli-common, which cannot import this module's `internal/`.
