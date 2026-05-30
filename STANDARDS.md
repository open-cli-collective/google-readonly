# google-readonly Standards Index

This file is an index, not a standalone Go style guide. Shared Open CLI standards live in `cli-common`; google-readonly-specific constraints live in this repository's local docs and structural tests.

## Repo-Local Standards

### Golden Principles

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/docs/golden-principles.md
Local convenience copy, if present: `docs/golden-principles.md`

Use this for enforced `gro` rules: non-destructive OAuth/API guardrails, interface-at-consumer command packages, `ClientFactory` injection, text-only resource leaves, dependency direction, context propagation, error wrapping, mocks, and shared test helpers.

### Architecture

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/docs/architecture.md
Local convenience copy, if present: `docs/architecture.md`

Use this for the package graph, command/client responsibilities, file naming conventions, and structural enforcement entrypoints.

### Adding a Google API Domain

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/docs/adding-a-domain.md
Local convenience copy, if present: `docs/adding-a-domain.md`

Use this when adding a Google API surface to `gro`. It names the OAuth scope, client package, command package, output, mock, fixture, registration, and verification steps that structural tests expect.

### Workspace Admin Setup

Source of truth: https://github.com/open-cli-collective/google-readonly/blob/main/WORKSPACE_ADMINS.md
Local convenience copy, if present: `WORKSPACE_ADMINS.md`

Use this for organization-managed OAuth client setup, Internal audience guidance, restricted-scope context, and distribution of the OAuth client JSON.

## Shared Open CLI Standards

### Repository Shape

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/repo-layout.md
Local convenience copy, if present: `../cli-common/docs/repo-layout.md`

Use this for the family-wide repository layout, required files, Makefile target names, lint config, Go version policy, branch settings, and commit hygiene.

### CI

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/ci.md
Local convenience copy, if present: `../cli-common/docs/ci.md`

Use this for shared CI behavior and how repository Makefile targets are consumed by GitHub workflows.

### Command Surface

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/command-surface.md
Local convenience copy, if present: `../cli-common/docs/command-surface.md`

Use this for command naming, positional arguments, flags, aliases, prompts, mutation safety, async command shape, and setup wizard behavior.

### Output and Rendering

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/output-and-rendering.md
Local convenience copy, if present: `../cli-common/docs/output-and-rendering.md`

Use this for text-first resource output, JSON carve-outs, output-shape flags, stream discipline, color, pagination, and presenter boundaries.

### Secrets

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-secrets.md
Local convenience copy, if present: `../cli-common/docs/working-with-secrets.md`

Use this for credential ingress, keyring storage, migration behavior, secret redaction, and no-leak tests.

### State

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-state.md
Local convenience copy, if present: `../cli-common/docs/working-with-state.md`

Use this for config/cache locations, credential references, cache freshness, state migration, and hermetic tests.

### Scriptability

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/scriptability.md
Local convenience copy, if present: `../cli-common/docs/scriptability.md`

Use this for non-interactive setup, env-bridge flags, health checks, OAuth browser handoff, and stdout/stderr behavior that scripts depend on.

### Release and Distribution

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/release.md
Local convenience copy, if present: `../cli-common/docs/release.md`

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/distribution.md
Local convenience copy, if present: `../cli-common/docs/distribution.md`

Use these for shared release and installation rules. Keep repository-specific release automation in the shared automation source, not in this file.

## Shared Automation

Source of truth: https://github.com/open-cli-collective/.github
Local convenience copy, if present: `../.github`

Use this for shared actions, reusable workflow implementations, and organization-level automation. Policy and conventions live in the shared standards docs above.

## Conflict Resolution

Local google-readonly docs define `gro`-specific constraints. `cli-common` docs define family-wide Open CLI standards. When a rule should apply to every CLI, update the shared source instead of copying the rule here.
