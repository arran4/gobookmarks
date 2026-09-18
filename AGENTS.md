# Agent instructions

## Commands

### `gobookmarks test verification template <subcommand range>`

**DEPRECATED:** Use `gobookmarks scenario serve <path>` instead for full executable application state verification. This old command exists only for backward compatibility and routes "complex" ranges to `scenarios/complex-bookmarks.txtar`.

### `gobookmarks scenario serve <path>`

This is the preferred way to construct realistic whole-application state for visual verification and testing. It spins up a disposable application instance using an in-memory database seeded by the given TXTAR scenario file.

Examples:
- `gobookmarks scenario serve cmd/gobookmarks/scenarios/complex-bookmarks.txtar`
- `gobookmarks scenario serve --port :8081 cmd/gobookmarks/scenarios/history.txtar`

**Important Note for Agents:**
- Use unit/template tests for isolated rendering.
- Use executable scenarios (via `scenario serve`) when a screenshot or review needs realistic whole-application state.
