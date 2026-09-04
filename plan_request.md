## Plan

1.  **Refactor `assets.go` Registry to support allowed list and live-mode cache policies.**
    -   Modify `Reload()` to only read public assets: `[]string{"main.css", "logo.png"}` instead of using `fs.WalkDir`. This prevents exposing the entire repository tree.
    -   Update `Registry` to support setting a custom cache-control string (or flag), e.g., via a `LiveMode bool` field or by wrapping it, so `ServeHTTP` can issue `Cache-Control: no-store` in live mode and `public, max-age=31536000, immutable` otherwise.

2.  **Fix `-tags=live` stale asset bug in `data_live.go`**
    -   In `data_live.go`, we can make `GetAssetRegistry()` return a struct implementing an `http.Handler` and providing an `AssetURL` method (we'll extract a simple interface or simply make `AssetURL` in `assets.go` call a dynamic function or use the global registry but reloaded).
    -   Alternatively, `data_live.go` can define a wrapper for `Registry` that calls `r.Reload()` on every `AssetURL` and `ServeHTTP` call, ensuring live reloading.
    -   Update `data_live.go`'s `GetMainCSSData()` and `GetFavicon()` to fetch from the file system the same way (or just read from the live registry).

3.  **Fix unnecessary duplicate embedding in `data_embedded.go`**
    -   Remove `//go:embed "main.css"` for `mainCSSData` and `//go:embed "logo.png"` for `faviconData`.
    -   Update `GetMainCSSData()` and `GetFavicon()` to fetch the raw bytes from `assetFS` (or from the `AssetRegistry` via logical name) to keep one authoritative source.

4.  **Fix compatibility route caching in `cmd/gobookmarks/serve.go`**
    -   Update the `/main.css` and `/favicon.ico` redirects to set `Cache-Control: no-cache` (or `no-store`) so clients aren't permanently stuck on old versions if the fallback routes are hit.
    -   Also make sure the raw byte fallback behavior (if `AssetURL` fails) sets `Cache-Control: no-store` or uses standard caching headers, rather than no headers.

5.  **Write and fix tests**
    -   Add regression tests in `assets_test.go` checking that non-allowlisted files (like `README.md` or a dummy `test.js`) cannot be served.
    -   Add a live mode test (in a new file or existing test with `+build live`) proving live modifications change the URL/hash and serve the correct bytes.
    -   Check compatibility route headers in `cmd/gobookmarks`.
    -   Run pre-commit instructions.
