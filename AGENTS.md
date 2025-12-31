# Agent instructions

## Commands

### `gobookmarks test verification template <subcommand range>`

This command is used for verification and creating screenshots.

Flags:
- `-data-from-json-file`: Optional. Load template data from a JSON file.
- `-serve <addr:port>`: Optional. Starts a HTTP server to serve the rendered template.
- `-out <file>`: Optional. Writes the output to a file.

**Note on Flag Ordering:** The CLI enforces strict flag ordering. Flags must be placed *before* the `<subcommand range>` argument.
Example: `gobookmarks test verification template -serve :8081 complex` (Correct)
Example: `gobookmarks test verification template complex -serve :8081` (Incorrect - flags will be ignored)

The `<subcommand range>` argument specifies which template/scenario to render. This mechanism allows agents to create specific test scenarios for UI verification.
