# Agent instructions

## Commands

### `gobookmarks test verification template <subcommand range>`

This command is used for verification and creating screenshots.

Flags:
- `-data-from-json-file`: Optional. Load template data from a JSON file.
- `-serve <addr:port>`: Optional. Starts a HTTP server to serve the rendered template.
- `-out <file>`: Optional. Writes the output to a file.

The `<subcommand range>` argument specifies which template/scenario to render. This mechanism allows agents to create specific test scenarios for UI verification.
