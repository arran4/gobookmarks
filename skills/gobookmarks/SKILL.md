# gobookmarks agent skill

This skill helps AI coding agents interact with the `gobookmarks` CLI and understand its architecture and operational behavior.

## Core Application Concepts

`gobookmarks` is a self-hosted personal landing page/bookmark manager backed by Git or SQL.

Key architectural characteristics:
- Multi-provider authentication (Database, GitHub, GitLab, Local Git).
- Plaintext bookmark storage format, parsed using custom delimiters (Category:, Tab:, Page:, Column, --).
- HTML Templates driven by Go's `html/template`, embedded via `//go:embed`.
- Stateful DB/Git connection pooling within `Provider` implementations.

## Common CLI Operations for Agents

When managing `gobookmarks` via CLI, keep these operational rules in mind:

### 1. Configuration
- Configuration is loaded via the `AppConfig` global or the `gobookmarks.Configuration` struct, primarily using `config.go`.
- Precedence: Environment Variables < Config File (`/etc/gobookmarks/config.json`) < Command Line Arguments.
- When generating configuration strings for the SQL provider, format for MySQL must include `multiStatements=true`.

### 2. Testing & Data Access
- When writing tests, ALWAYS initialize the global config (e.g., `AppConfig = gobookmarks.Configuration{...}`) instead of manipulating deprecated global variables.
- Write operations (e.g., `AddPage`, `EditBookmark`) must invalidate the request cache using `invalidateRequestCache`. Ensure integration tests cover cache invalidation properly.

### 3. File Updates and Safety
- Update commands should generally be idempotent or have strict locking if operating on the git tree.
- When parsing raw bookmarks, rely on tests in `testdata/txtar/` to ensure text extraction boundaries (`ExtractCategoryByIndex`) don't unintentionally eat structural markers like `Column` or `--`.

## Pitfalls and Traps
- **Do not mock incomplete Providers:** When writing provider tests, if you create a mock `Provider`, it MUST implement all interface methods or the app will panic upon `RegisterProvider()`.
- **Do not use `log.Fatalf` in goroutines:** It immediately exits the program with code 1, bypassing the main thread's error handling. Return errors over channels instead.
- **Form Submissions:** Optimistic locking for updates (e.g., editing bookmarks) requires `branch`, `ref`, and `sha` hidden fields in POST submissions. Omitting these will cause updates to fail with validation errors.

Use this knowledge to safely scaffold features, write tests, and manage the `gobookmarks` lifecycle.
