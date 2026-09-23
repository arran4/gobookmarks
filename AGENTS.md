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
- A scenario may authenticate through `AuthProvider` while rendering data from its disposable `StorageProvider`; `scenario serve` applies that storage choice only to its request context. External login fixtures use the manifest `AuthUser` and reject undeclared outbound requests.

### Caching Architecture

- The application uses a strictly request-scoped cache (`requestCache`) bound to `CoreData` via `CoreAdderMiddleware`.
- The cache deduplicates `GetBookmarks()` calls within a single HTTP request, which frequently occur during template rendering (e.g., retrieving tab names and lists multiple times per page). A counted illustrative template test leveraging genuine `FuncMap` helpers proves that un-cached template reads hit providers exactly 3 times per execution, whereas the request cache correctly deduplicates this to exactly 1 call while matching the identical rendered output.
- The request-scoped cache naturally operates with no TTL; `CoreAdderMiddleware` creates a fresh `CoreData` instance for each normal sequential request. This context ordinarily becomes unreachable and is garbage collected after request handling. A single `CoreData` instance maintains stable provider, user, and token contexts since application paths do not share `CoreData` concurrently. However, the cache is explicitly keyed by `<user>|<ref>|<providerName>` to isolate environments where `Provider` may be overridden mid-request (such as within execution scenarios).
- Providers use SHA-based identifiers for basic change detection, but Git semantics dictate this SHA represents the history state and does not guarantee strict monotonic concurrency on its own. The underlying systems process writes with varying levels of safety:
    - **GitHub**: If `expectSHA` is provided, it is compared against a *freshly fetched* `contents.SHA`. It then passes the newly fetched `contents.SHA` to `UpdateFile` (skipping the preliminary comparison if empty). This creates a read/check/write window that is not strictly atomic against racing commits.
    - **GitLab**: Submits `LastCommitID` back to the server API, without universal proof of branch-head atomicity.
    - **Local Git**: Stale versions fail natively on `sha mismatch` only if `expectSHA` is non-empty. It checks `expectSHA` against `head.Hash().String()` manually. This creates a read/check/write window and is not strictly atomic.
    - **SQL**: Conditional updates are evaluated only when `expectSHA != ""` and `curSha.Valid`. It uses an ordinary database transaction, but distinguishes transaction atomicity from atomic version comparison by only performing a standard `SELECT` check before inserting/updating, lacking a `FOR UPDATE` lock.
- There is no cross-request or process-wide cache. It was intentionally removed (#257) to avoid complex staleness bugs, as bookmarks are updated externally and most providers lack efficient push notifications for invalidation. Any future cross-request caching should only be implemented as a separate follow-up if strictly evidenced.
