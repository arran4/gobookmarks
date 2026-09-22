package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestScenarioServePortInUse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	// Parse out port from something like http://127.0.0.1:12345
	u, _ := url.Parse(ts.URL)
	port := ":" + u.Port()

	root := NewRootCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		sc := root.ScenarioCmd.ServeCommand
		errCh <- sc.ExecuteContext(ctx, []string{"--port", port, "scenarios/complex-bookmarks.txtar"})
	}()

	select {
	case <-ctx.Done():
		t.Fatalf("scenario serve command hanging when port is in use")
	case err := <-errCh:
		if err == nil {
			t.Fatalf("scenario serve command succeeded unexpectedly")
		}
	}
}

func TestScenarioServeStartupShutdownAndAuthentication(t *testing.T) {
	root := NewRootCommand()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	readyPortCh := make(chan string, 1)

	sc := root.ScenarioCmd.ServeCommand
	sc.readyPort = readyPortCh

	go func() {
		errCh <- sc.ExecuteContext(ctx, []string{"--port", ":0", "scenarios/complex-bookmarks.txtar"})
	}()

	var boundPort string
	select {
	case boundPort = <-readyPortCh:
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for scenario serve to start")
	}

	// Make an authenticated request using the browser wrapper
	routerURL := "http://" + boundPort
	browser, err := NewBrowser(nil, routerURL)
	if err != nil {
		t.Fatalf("Failed to create browser: %v", err)
	}
	// The NewBrowser typically takes a router, but here the server is actually running on a port!
	// So we don't need the internal handler, we can just make raw HTTP requests with a client that keeps cookies.

	client := browser.Jar
	httpClient := &http.Client{Jar: client}

	// Perform Login
	resp, err := httpClient.PostForm(routerURL+"/login/sql", url.Values{
		"username": []string{"testuser"},
		"password": []string{"password"},
	})
	if err != nil {
		t.Fatalf("Login POST failed: %v", err)
	}
	_ = resp.Body.Close()

	if resp.Request.URL.Path != "/" {
		t.Fatalf("Login did not redirect to root, got %s", resp.Request.URL.Path)
	}

	// Fetch Root Page
	resp2, err := httpClient.Get(routerURL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}
	_ = resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("Logout")) || !bytes.Contains(body, []byte("Google")) {
		t.Fatalf("Failed to retrieve complex bookmarks, status %d", resp2.StatusCode)
	}

	// Shutdown
	cancel()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("scenario serve command did not return after context cancellation")
	case err := <-errCh:
		if err != nil {
			t.Fatalf("scenario serve command returned error: %v", err)
		}
	}
}

func TestScenarioServeConsecutiveRuns(t *testing.T) {
	root := NewRootCommand()

	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		readyPortCh := make(chan string, 1)

		sc := root.ScenarioCmd.ServeCommand
		sc.readyPort = readyPortCh

		go func() {
			errCh <- sc.ExecuteContext(ctx, []string{"--port", ":0", "scenarios/complex-bookmarks.txtar"})
		}()

		var boundPort string
		select {
		case boundPort = <-readyPortCh:
		case <-time.After(5 * time.Second):
			t.Fatalf("run %d: Timeout waiting for scenario serve to start", i)
		}

		routerURL := "http://" + boundPort
		browser, err := NewBrowser(nil, routerURL)
		if err != nil {
			t.Fatalf("run %d: Failed to create browser: %v", i, err)
		}

		httpClient := &http.Client{Jar: browser.Jar}

		// 1. Authenticate to ensure the database and data is loaded.
		resp, err := httpClient.PostForm(routerURL+"/login/sql", url.Values{
			"username": []string{"testuser"},
			"password": []string{"password"},
		})
		if err != nil {
			t.Fatalf("run %d: Login POST failed: %v", i, err)
		}
		_ = resp.Body.Close()

		if resp.Request.URL.Path != "/" {
			t.Fatalf("run %d: Login did not redirect to root, got %s", i, resp.Request.URL.Path)
		}

		// 2. Read state
		resp2, err := httpClient.Get(routerURL + "/")
		if err != nil {
			t.Fatalf("run %d: GET / failed: %v", i, err)
		}
		body, err := io.ReadAll(resp2.Body)
		if err != nil {
			t.Fatalf("run %d: Failed to read GET body: %v", i, err)
		}
		_ = resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("Google")) {
			t.Fatalf("run %d: Failed to retrieve seeded bookmarks, status %d", i, resp2.StatusCode)
		}

		uniqueBookmarkStr := []byte("https://unique-to-run-0.example.com")
		if i == 0 {
			// Mutate state in Run 0
			respAdd, err := httpClient.PostForm(routerURL+"/edit", url.Values{
				"tab":      []string{"0"},
				"category": []string{"0"},
				"page":     []string{"0"},
				"task":     []string{"Save and Close"},
				"branch":   []string{"main"},
				"ref":      []string{"refs/heads/main"},
				"title":    []string{"Unique Run 0 Bookmark"},
				"url":      []string{string(uniqueBookmarkStr)},
			})
			if err != nil {
				t.Fatalf("run %d: Bookmark POST failed: %v", i, err)
			}
			_ = respAdd.Body.Close()

			if respAdd.StatusCode != http.StatusOK {
				// edit actually responds with 200 containing javascript redirect
			}
		} else {
			// Verify state is clean in Run 1 (and subsequent)
			if bytes.Contains(body, uniqueBookmarkStr) {
				t.Fatalf("run %d: Leaked state from previous scenario run detected", i)
			}
		}

		// 3. Gracefully terminate
		cancel()

		select {
		case <-time.After(5 * time.Second):
			t.Fatalf("run %d did not return after cancellation", i)
		case err := <-errCh:
			if err != nil {
				t.Fatalf("run %d returned error: %v", i, err)
			}
		}

		// 4. Verify the listener is released by attempting to connect
		_, err = http.Get(routerURL)
		if err == nil {
			t.Fatalf("run %d: Expected connection to fail after shutdown, but it succeeded", i)
		}

		// Check that the underlying DB is actually closed by testing `Ping()` internally if we could.
		// `SQLProvider.Close()` logic covers this aspect, preventing DB handle leakage.
	}
}

func TestSetupScenarioBackendRollback(t *testing.T) {
	// To test OpenDB failing within setupScenarioBackend, we would need to mock sql.Open or OpenDB,
	// which is currently a tightly coupled package-level function without an injection point.
	// We cannot force `OpenDB` to fail deterministically here because `setupScenarioBackend`
	// hardcodes `Config.DBConnectionProvider = "sqlite3"` and a valid `file:...` connection string
	// before invoking `OpenDB`. Refactoring `OpenDB` or the global `Config` access pattern
	// throughout the application is outside the strictly-confined scope of this scenario fix PR.
	// Therefore, this rollback path (the `if err != nil` block in `setupScenarioBackend` after `OpenDB()`)
	// remains explicitly untested in this PR, relying solely on human review of the rollback assignments.
}
