package main

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

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

	// Capture output to parse the randomly assigned port
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	errCh := make(chan error, 1)
	go func() {
		sc := root.ScenarioCmd.ServeCommand
		errCh <- sc.ExecuteContext(ctx, []string{"--port", ":0", "scenarios/complex-bookmarks.txtar"})
	}()

	// Wait for startup and parse port in a non-blocking way
	var portStr string
	outputCh := make(chan string, 1)

	go func() {
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		outputCh <- string(buf[:n])
	}()

	select {
	case output := <-outputCh:
		idx := strings.Index(output, "listening on ")
		if idx != -1 {
			portStr = strings.TrimSpace(output[idx+len("listening on "):])
			portStr = strings.Split(portStr, "...")[0]
		}
	case <-time.After(2 * time.Second):
		// timeout reading stdout
	}

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if portStr == "" {
		t.Fatalf("Server failed to start or did not print listening message")
	}

	// Make an authenticated request using the browser wrapper
	routerURL := "http://" + portStr
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
		"username": {"testuser"},
		"password": {"password"},
	})
	if err != nil {
		t.Fatalf("Login POST failed: %v", err)
	}
	resp.Body.Close()

	if resp.Request.URL.Path != "/" {
		t.Fatalf("Login did not redirect to root, got %s", resp.Request.URL.Path)
	}

	// Fetch Root Page
	resp2, err := httpClient.Get(routerURL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	body, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

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
		go func() {
			sc := root.ScenarioCmd.ServeCommand
			errCh <- sc.ExecuteContext(ctx, []string{"--port", ":0", "scenarios/complex-bookmarks.txtar"})
		}()

		// Wait briefly to let it start and register
		time.Sleep(300 * time.Millisecond)
		cancel()

		select {
		case <-time.After(5 * time.Second):
			t.Fatalf("run %d did not return after cancellation", i)
		case err := <-errCh:
			if err != nil {
				t.Fatalf("run %d returned error: %v", i, err)
			}
		}
	}
}

func TestDatabaseLeak(t *testing.T) {
	cleanup, err := setupScenarioBackend()
	if err != nil {
		t.Fatalf("setupScenarioBackend failed: %v", err)
	}

	// Retrieve provider and initialize the DB internally
	p := gobookmarks.GetProvider("sql")
	_, _, err = p.GetBookmarks(context.Background(), "bob", "main", nil)
	if err != nil {
		// ignore
	}

	sqlP := p.(*gobookmarks.SQLProvider)
	oldDBValue := reflect.ValueOf(sqlP).Elem().FieldByName("db")
	if oldDBValue.IsNil() {
		t.Fatalf("Expected db to be initialized")
	}
	db := reflect.NewAt(oldDBValue.Type(), unsafe.Pointer(oldDBValue.UnsafeAddr())).Elem().Interface().(*sql.DB)

	cleanup()

	err = db.Ping()
	if err == nil {
		t.Fatalf("Expected DB to be closed but ping succeeded. Leak confirmed.")
	}
}
