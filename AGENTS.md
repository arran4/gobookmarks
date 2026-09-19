# Agent instructions

## Commands

### `gobookmarks test verification template <subcommand range>`

This command is used for template-level verification and isolated UI rendering testing.

Flags:
- `-data-from-json-file`: Optional. Load template data from a JSON file.
- `-serve <addr:port>`: Optional. Starts a HTTP server to serve the rendered template.
- `-out <file>`: Optional. Writes the output to a file.

**Note on Flag Ordering:** The CLI enforces strict flag ordering. Flags must be placed *before* the `<subcommand range>` argument.
Example: `gobookmarks test verification template -serve :8081 complex` (Correct)

### `gobookmarks scenario serve <path>`

This is the preferred way to construct realistic whole-application state for visual verification, regression testing, and screenshots requiring integration-level setup. It spins up a disposable application instance using an in-memory database (or mocked backend) seeded by the given TXTAR scenario file, and wires up the *real application router and handlers*.

Examples:
- `gobookmarks scenario serve cmd/gobookmarks/scenarios/complex-bookmarks.txtar`
- `gobookmarks scenario serve --port :8081 cmd/gobookmarks/scenarios/history.txtar`

**Important Note for Agents:**
- Use `test verification template` (isolated unit/template rendering tests) for template-specific work.
- Use executable scenarios (via `scenario serve`) when a screenshot or review needs realistic whole-application state.
- `scenario apply` is a disposable rehearsal: it validates and seeds temporary state, then removes it when the command exits. It never writes a configured backend. Use `scenario serve` and log in through the normal route to inspect seeded state in a browser.
