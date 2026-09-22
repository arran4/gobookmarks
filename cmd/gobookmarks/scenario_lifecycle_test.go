package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/arran4/gobookmarks"
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
			// Mutate state in Run 0 by updating the raw text entirely
			newText := "Tab: Tab 1\nPage: Page 1\nCategory: Search Engines\n" + string(uniqueBookmarkStr) + " Unique\n"

			respAdd, err := httpClient.PostForm(routerURL+"/edit", url.Values{
				"task":   []string{"Save and Close"},
				"branch": []string{"main"},
				"ref":    []string{"refs/heads/main"},
				"text":   []string{newText},
			})
			if err != nil {
				t.Fatalf("run %d: Bookmark POST failed: %v", i, err)
			}

			bodyAdd, readErr := io.ReadAll(respAdd.Body)
			_ = respAdd.Body.Close()
			if readErr != nil {
				t.Fatalf("run %d: Failed to read edit response body: %v", i, readErr)
			}

			if respAdd.StatusCode != http.StatusOK {
				t.Fatalf("run %d: Bookmark POST returned unexpected status %d: %s", i, respAdd.StatusCode, bodyAdd)
			}

			// Verify mutation is present
			respVerify, err := httpClient.Get(routerURL + "/")
			if err != nil {
				t.Fatalf("run %d: GET / after edit failed: %v", i, err)
			}
			verifyBody, readErr := io.ReadAll(respVerify.Body)
			_ = respVerify.Body.Close()
			if readErr != nil {
				t.Fatalf("run %d: Failed to read verify response body: %v", i, readErr)
			}

			if !bytes.Contains(verifyBody, uniqueBookmarkStr) {
				t.Fatalf("run %d: Mutation was not persisted! Body did not contain unique string.", i)
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

		// 4. Verify the listener is released explicitly by rebinding
		ln, err := net.Listen("tcp", boundPort)
		if err != nil {
			t.Fatalf("run %d: Expected port %s to be released, but could not bind it: %v", i, boundPort, err)
		}
		_ = ln.Close()

		// Check that the underlying DB is actually closed by testing `Ping()` internally if we could.
		// `SQLProvider.Close()` logic covers this aspect, preventing DB handle leakage.
	}
}

func TestProviderUnregisterCleanup(t *testing.T) {
	// Capture original process-global state
	originalProvider := gobookmarks.GetProvider("github")
	originalOrder := append([]string{}, gobookmarks.Config.ProviderOrder...)

	// Register unconditional restoration of the original state
	t.Cleanup(func() {
		if originalProvider != nil {
			gobookmarks.RegisterProvider(originalProvider)
		} else {
			gobookmarks.UnregisterProvider("github")
		}
		gobookmarks.SetProviderOrder(originalOrder)
	})

	// First unregister github if it exists to ensure it's absent
	gobookmarks.UnregisterProvider("github")

	if gobookmarks.GetProvider("github") != nil {
		t.Fatalf("github provider should be unregistered initially")
	}

	cleanup, err := setupScenarioBackend()
	if err != nil {
		t.Fatalf("setupScenarioBackend failed: %v", err)
	}

	// Register unconditional teardown of the scenario backend if we fail midway
	t.Cleanup(func() {
		if cleanup != nil {
			cleanup()
			cleanup = nil // prevent double-cleaning
		}
	})

	// Verify setupScenarioBackend registers it
	if gobookmarks.GetProvider("github") == nil {
		t.Fatalf("github provider should be registered by scenario setup")
	}

	// Trigger cleanup manually to assert its behavior
	cleanup()
	cleanup = nil

	// Verify cleanup completely unregisters it again because it was originally absent
	if gobookmarks.GetProvider("github") != nil {
		t.Fatalf("github provider should be unregistered after cleanup")
	}
}
