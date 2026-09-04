package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	gb "github.com/arran4/gobookmarks"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func TestRunHandlerChain_UserErrorRedirect(t *testing.T) {
	gb.Config.SessionName = "testsess"
	gb.SessionStore = sessions.NewCookieStore([]byte("secret"))

	req := httptest.NewRequest("GET", "/submit", nil)
	req.Header.Set("Referer", "/form")
	ctx := context.WithValue(req.Context(), gb.ContextValues("coreData"), &gb.CoreData{})
	req = req.WithContext(ctx)

	h := runHandlerChain(func(_ http.ResponseWriter, _ *http.Request) error {
		return gb.NewUserError("bad input", errors.New("invalid"))
	})

	w := httptest.NewRecorder()
	h(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if !strings.Contains(loc, "error=bad+input") {
		t.Fatalf("redirect missing error param: %s", loc)
	}
}

func TestRunTemplate_BufferedError(t *testing.T) {
	gb.Config.SessionName = "testsess"
	gb.SessionStore = sessions.NewCookieStore([]byte("secret"))
	gb.Config.DBConnectionProvider = ""

	req := httptest.NewRequest("GET", "/", nil)
	sess, _ := gb.SessionStore.New(req, gb.Config.SessionName)
	sess.Values["GithubUser"] = &gb.User{Login: "user"}
	sess.Values["Token"] = &oauth2.Token{}
	ctx := context.WithValue(req.Context(), gb.ContextValues("session"), sess)
	ctx = context.WithValue(ctx, gb.ContextValues("provider"), "sql")
	ctx = context.WithValue(ctx, gb.ContextValues("coreData"), &gb.CoreData{UserRef: "user"})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	runTemplate("mainPage.gohtml")(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Database error") {
		t.Fatalf("expected database error message, got %q", body)
	}
	if strings.Count(body, "<!DOCTYPE html>") != 1 {
		t.Fatalf("unexpected partial content: %q", body)
	}
	if strings.Contains(body, "tab-list") {
		t.Fatalf("unexpected partial page content: %q", body)
	}
}

func TestLoadConfigUsesExternalURL(t *testing.T) {
	rc := NewRootCommand()

	t.Setenv("EXTERNAL_URL", "http://example.com/app")

	if err := rc.loadConfig(); err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if rc.cfg.ExternalURL != "http://example.com/app" {
		t.Fatalf("external url not loaded from env: %q", rc.cfg.ExternalURL)
	}
}

func TestLoadConfig_EnvPrecedence(t *testing.T) {
	t.Setenv("LOCAL_GIT_PATH", "/env/path")
	t.Setenv("SESSION_NAME", "env_session")
	t.Setenv("GBM_CSS_COLUMNS", "0") // Documented contract: any non-empty value sets it to true
	t.Setenv("GBM_NO_FOOTER", "true")
	t.Setenv("GBM_DEV_MODE", "false")
	t.Setenv("COMMITS_PER_PAGE", "50")

	jsonConfig := `{"local_git_path": "/json/path", "css_columns": false, "no_footer": false, "dev_mode": true, "commits_per_page": 0, "session_name": ""}`

	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"
	if err := os.WriteFile(configPath, []byte(jsonConfig), 0644); err != nil {
		t.Fatal(err)
	}

	rc := NewRootCommand()
	rc.ConfigPath = configPath // JSON config should override env
	if err := rc.loadConfig(); err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if rc.cfg.LocalGitPath != "/json/path" {
		t.Fatalf("expected /json/path, got %q", rc.cfg.LocalGitPath)
	}
	if rc.cfg.SessionName != "" {
		t.Fatalf("expected empty string from json override, got %q", rc.cfg.SessionName)
	}
	if rc.cfg.CommitsPerPage != 0 {
		t.Fatalf("expected 0 from json override, got %d", rc.cfg.CommitsPerPage)
	}
	if rc.cfg.CSSColumns != false {
		t.Fatalf("expected CSSColumns false from json override, got %v", rc.cfg.CSSColumns)
	}
	if rc.cfg.NoFooter != false {
		t.Fatalf("expected NoFooter false from json override, got %v", rc.cfg.NoFooter)
	}
	if rc.cfg.DevMode == nil || *rc.cfg.DevMode != true {
		t.Fatalf("expected DevMode true from json override, got %v", rc.cfg.DevMode)
	}

	serveCmd, _ := rc.NewServeCommand()
	// Command line args should override json
	if err := serveCmd.Flags.Parse([]string{"-local-git-path=/flag/path", "-css-columns=true", "-dev-mode=false"}); err != nil {
		t.Fatal(err)
	}

	// Simulate Execute behavior updating cfg
	if serveCmd.LocalGitPath.set {
		rc.cfg.LocalGitPath = serveCmd.LocalGitPath.value
	}
	if serveCmd.CSSColumns.set {
		rc.cfg.CSSColumns = serveCmd.CSSColumns.value
	}
	if serveCmd.DevMode.set {
		rc.cfg.DevMode = &serveCmd.DevMode.value
	}

	if rc.cfg.LocalGitPath != "/flag/path" {
		t.Fatalf("expected /flag/path, got %q", rc.cfg.LocalGitPath)
	}
	if rc.cfg.CSSColumns != true {
		t.Fatalf("expected CSSColumns true from flag override, got %v", rc.cfg.CSSColumns)
	}
	if rc.cfg.DevMode == nil || *rc.cfg.DevMode != false {
		t.Fatalf("expected DevMode false from flag override, got %v", rc.cfg.DevMode)
	}

	// Update global config as serve would do, and check if provider is configured
	gb.Config = rc.cfg
	provs := gb.ConfiguredProviderNames()
	foundGit := false
	for _, p := range provs {
		if p == "git" {
			foundGit = true
			break
		}
	}
	if !foundGit {
		t.Fatalf("expected 'git' provider to be configured since LocalGitPath is set")
	}
}
