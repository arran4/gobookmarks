package gobookmarks

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func setupHandlerTest(t *testing.T, initial string) (GitProvider, string, context.Context) {
	tmp := t.TempDir()
	LocalGitPath = tmp
	p := GitProvider{}
	user := "alice"
	if err := p.CreateRepo(context.Background(), user, nil, RepoName); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	if err := p.CreateBookmarks(context.Background(), user, nil, "main", initial); err != nil {
		t.Fatalf("CreateBookmarks: %v", err)
	}
	SessionName = "testsess"
	SessionStore = sessions.NewCookieStore([]byte("secret"))
	sessReq := httptest.NewRequest("GET", "/", nil)
	sess, _ := getSession(httptest.NewRecorder(), sessReq)
	sess.Values["GithubUser"] = &User{Login: user}
	sess.Values["Token"] = &oauth2.Token{}
	ctx := context.WithValue(sessReq.Context(), ContextValues("session"), sess)
	ctx = context.WithValue(ctx, ContextValues("provider"), "git")
	return p, user, ctx
}

func TestAddTabHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\n")
	values := url.Values{"name": {"New"}}
	req := httptest.NewRequest("POST", "/addTab?index=1&branch=main&ref=refs/heads/main", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := AddTabHandler(w, req); err != ErrHandled {
		t.Fatalf("AddTabHandler: %v", err)
	}
	got, _, err := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	tabs := PreprocessBookmarks(got)
	if len(tabs) != 2 || tabs[1].Name != "New" {
		t.Fatalf("tab not added correctly: %#v", tabs)
	}
}

func TestAddPageHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\n")
	form := url.Values{"name": {"P2"}}
	req := httptest.NewRequest("POST", "/addPage?tab=0&index=1&branch=main&ref=refs/heads/main", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := AddPageHandler(w, req); err != ErrHandled {
		t.Fatalf("AddPageHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	pages := PreprocessBookmarks(got)[0].Pages
	if len(pages) != 2 || pages[1].Name != "P2" {
		t.Fatalf("page not added correctly: %#v", pages)
	}
}

func TestAddCategoryHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\n")
	req := httptest.NewRequest("POST", "/addCategory?tab=0&page=0&block=0&col=0&index=1&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := AddCategoryHandler(w, req); err != ErrHandled {
		t.Fatalf("AddCategoryHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	cats := PreprocessBookmarks(got)[0].Pages[0].Blocks[0].Columns[0].Categories
	if len(cats) != 2 {
		t.Fatalf("category not added: got %d", len(cats))
	}
}
