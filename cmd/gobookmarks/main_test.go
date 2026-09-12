package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	gb "github.com/arran4/gobookmarks"

	"golang.org/x/oauth2"
)

type MockEnv struct {
	Env map[string]string
	Uid int
}

func (m MockEnv) Getenv(key string) string       { return m.Env[key] }
func (m MockEnv) Setenv(key, value string) error { m.Env[key] = value; return nil }
func (m MockEnv) Geteuid() int                   { return m.Uid }

type MockFileReader struct {
	Files map[string][]byte
}

func (m *MockFileReader) ReadFile(name string) ([]byte, error) {
	if data, ok := m.Files[name]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}
func (m *MockFileReader) Open(name string) (io.ReadCloser, error) { return nil, os.ErrNotExist }
func (m *MockFileReader) Stat(name string) (os.FileInfo, error) {
	if _, ok := m.Files[name]; ok {
		// returning nil fileinfo is problematic if accessed, but we just need err == nil for fileExists
		return nil, nil
	}
	return nil, os.ErrNotExist
}

func TestRunHandlerChain_UserErrorRedirect(t *testing.T) {
	gb.Config.SessionName = "testsess"
	gb.SessionStore = gb.InitSessionStore([]byte("secret"))

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
	gb.SessionStore = gb.InitSessionStore([]byte("secret"))
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

	env := MockEnv{Env: map[string]string{"EXTERNAL_URL": "http://example.com/app"}, Uid: 1000}
	if err := rc.loadConfig(env); err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if rc.cfg.ExternalURL != "http://example.com/app" {
		t.Fatalf("external url not loaded from env: %q", rc.cfg.ExternalURL)
	}
}

func TestLoadConfig_EnvOnlyProvider(t *testing.T) {
	rc := NewRootCommand()
	env := MockEnv{Env: map[string]string{"LOCAL_GIT_PATH": "/env/path/for/git"}, Uid: 1000}
	if err := rc.loadConfig(env); err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	// Ensure that LocalGitPath actually populates
	if rc.cfg.LocalGitPath != "/env/path/for/git" {
		t.Fatalf("expected /env/path/for/git, got %q", rc.cfg.LocalGitPath)
	}

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
		t.Fatalf("expected 'git' provider to be configured from env variable LOCAL_GIT_PATH alone")
	}
}

func TestLoadConfig_EnvPrecedence(t *testing.T) {
	env := MockEnv{Env: map[string]string{
		"LOCAL_GIT_PATH":   "/env/path",
		"SESSION_NAME":     "env_session",
		"GBM_CSS_COLUMNS":  "0",
		"GBM_NO_FOOTER":    "true",
		"GBM_DEV_MODE":     "false",
		"COMMITS_PER_PAGE": "50",
	}, Uid: 1000}

	jsonConfig := `{"local_git_path": "/json/path", "css_columns": false, "no_footer": false, "dev_mode": true, "commits_per_page": 0, "session_name": ""}`
	fs := &MockFileReader{Files: map[string][]byte{"/config.json": []byte(jsonConfig)}}
	_ = fs

	rc := NewRootCommand()
	rc.ConfigPath = "/config.json"
	if err := rc.loadConfig(env, fs); err != nil {
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

func TestDefaultSessionKeyPath(t *testing.T) {
	envRoot := MockEnv{Env: map[string]string{"XDG_STATE_HOME": "/xdg_state", "HOME": "/home/root"}, Uid: 0}
	fsRoot := &MockFileReader{Files: map[string][]byte{"/var/lib/gobookmarks/session.key": []byte("key")}}

	path := gb.DefaultSessionKeyPath(false, envRoot, fsRoot)
	if path != "/var/lib/gobookmarks/session.key" {
		t.Errorf("Expected root path to be /var/lib/gobookmarks/session.key, got %s", path)
	}

	envUser := MockEnv{Env: map[string]string{"XDG_STATE_HOME": "/xdg_state", "HOME": "/home/user"}, Uid: 1000}
	fsUser := &MockFileReader{Files: map[string][]byte{"/xdg_state/gobookmarks/session.key": []byte("key")}}

	path = gb.DefaultSessionKeyPath(false, envUser, fsUser)
	if path != "/xdg_state/gobookmarks/session.key" {
		t.Errorf("Expected user path to be /xdg_state/gobookmarks/session.key, got %s", path)
	}
}

func TestLoadEnvFile(t *testing.T) {
	env := MockEnv{Env: map[string]string{"EXISTING_KEY": "existing_value"}, Uid: 1000}

	envContent := `
# A comment
NEW_KEY=new_value
EXISTING_KEY=overridden_value
BLANK=

INVALID_LINE
`
	fs := &MockFileReader{Files: map[string][]byte{"/test.env": []byte(envContent)}}

	err := gb.LoadEnvFile("/test.env", env, fs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if env.Env["NEW_KEY"] != "new_value" {
		t.Errorf("Expected NEW_KEY to be new_value, got %s", env.Env["NEW_KEY"])
	}

	if env.Env["EXISTING_KEY"] != "existing_value" {
		t.Errorf("Expected EXISTING_KEY to retain its original value, got %s", env.Env["EXISTING_KEY"])
	}

	if env.Env["BLANK"] != "" {
		t.Errorf("Expected BLANK to be empty, got %s", env.Env["BLANK"])
	}
}
