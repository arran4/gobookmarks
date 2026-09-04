1. **Fix `-tags=live` asset refreshing**
   - We need `assets.go`'s `GetAsset` and `AssetURL` to serve up-to-date data for live builds. Currently `data_live.go` instantiates a registry over `os.DirFS` using `sync.Once`.
   - Modify `data_live.go` to provide a custom `Registry` implementation or have a `LiveAssetRegistry` that reloads data dynamically.
   - Wait, `assets.go` contains `AssetRegistry` logic. We could add a `isLive` flag to the registry or make `Reload()` happen on each request if in live mode. Or better, we can inject a `LiveAssetRegistry` type in `data_live.go` that implements an interface, or just call `Reload()` inside `GetAsset`/`AssetURL` for live mode.
   - We can modify `data_live.go` to have `GetAssetRegistry()` return something that wraps `Registry` and reloads it, or modify `Registry` to support live mode (e.g., `reg.liveMode = true` and in `GetAsset` / `AssetURL`, it does `Reload()`).

2. **Do not expose the repository tree as assets**
   - In `assets.go:Reload()`, the current code walks the filesystem and adds everything to the registry, except specific skipped directories.
   - The requirement is: "Restrict asset exposure to the intended public files: - main.css - logo.png". Add regression test.
   - Change the behavior in `Reload()` to only add a known list of files: `[]string{"main.css", "logo.png"}`. Read exactly those files. No need to walk the tree.

3. **Preserve strict production fingerprint semantics**
   - The URLs must be `/assets/main.<sha256>.css`.
   - Ensure cache-control is `public, max-age=31536000, immutable`.
   - Return 404 for unknown assets.

4. **Review compatibility routes**
   - In `serve.go` (and `test_verification_template_command.go`), `/main.css` and `/favicon.ico` currently redirect to `AssetURL()`. But they don't set cache control on the redirect, which means the redirect response might not be cached safely or might be cached too aggressively? Wait, `http.StatusFound` (302) is not cached by default, but maybe we should add `Cache-Control: no-cache` for the redirect? "the redirect response itself should not be cached in a way that permanently points clients at an old fingerprint" - `302 Found` without explicit cache headers is usually okay but `no-store` or `no-cache` is safer. "fallback behavior should have a safe cache policy". Also, the fallback if asset not found writes bytes directly with no cache-control. We should set appropriate cache control on these.

5. **Avoid unnecessary duplicate embedding**
   - `data_embedded.go` currently embeds `main.css` and `logo.png` into `assetFS`, and ALSO into `mainCSSData` and `faviconData`.
   - The requirement says: "simplify this so there is one authoritative embedded asset source while retaining compatibility APIs such as GetMainCSSData() / GetFavicon() if callers still need them."
   - We can remove the `//go:embed "main.css"` for `mainCSSData` and just have `GetMainCSSData()` read from `assetFS` (or get it from `AssetRegistry`).

6. **Check old fingerprint / deployment semantics**
   - We just need to make sure we serve 404 for old hashes. (Already doing this).

7. **Review the whole new asset implementation for adjacent correctness issues**
   - Concurrency/race around registry reloads. In `assets.go` `Reload()` takes a lock and updates maps, but it walks the tree *without* a lock first (which is good).
   - "HTTP method and content-type handling": `serve.go` routing for assets uses `.Methods("GET")` but what if `ServeHTTP` is called directly?
   - "whether returned Asset values can be mutated unsafely": `GetAsset` returns a pointer to `Asset` which contains `[]byte`. Someone could mutate `Asset.Bytes`. We should perhaps return a copy or just not export `Asset` fields. `ServeContent` does `bytes.NewReader(asset.Bytes)` which is safe.
   - "live responses use an appropriate non-stale development cache policy": In `assets.go`, `ServeHTTP` hardcodes `Cache-Control: public, max-age=31536000, immutable`. If it's live mode, it should be `Cache-Control: no-store`.

8. **Tests**
   - Write tests for live mode asset content changes.
   - Write tests for non-public repository files not being exposed.
   - Compatibility route tests.
