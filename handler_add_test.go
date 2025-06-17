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
	ctx = context.WithValue(ctx, ContextValues("coreData"), &CoreData{})
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

func TestAddTabHandlerInvalid(t *testing.T) {
	p, _, ctx := setupHandlerTest(t, "Category: A\n")
	values := url.Values{"name": {"Bad"}}
	req := httptest.NewRequest("POST", "/addTab?index=5&branch=main&ref=refs/heads/main", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	err := AddTabHandler(w, req)
	if err == nil || err == ErrHandled {
		t.Fatalf("expected error got %v", err)
	}
	// ensure repo unchanged
	got, _, _ := p.GetBookmarks(context.Background(), "alice", "refs/heads/main", nil)
	tabs := PreprocessBookmarks(got)
	if len(tabs) != 1 {
		t.Fatalf("tabs changed unexpectedly")
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

func TestAddPageHandlerInvalid(t *testing.T) {
	p, _, ctx := setupHandlerTest(t, "Category: A\n")
	form := url.Values{"name": {"P"}}
	// invalid index beyond pages length
	req := httptest.NewRequest("POST", "/addPage?tab=0&index=5&branch=main&ref=refs/heads/main", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	err := AddPageHandler(w, req)
	if err == nil || err == ErrHandled {
		t.Fatalf("expected error got %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), "alice", "refs/heads/main", nil)
	pages := PreprocessBookmarks(got)[0].Pages
	if len(pages) != 1 {
		t.Fatalf("pages changed unexpectedly")
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
func TestDeleteTabHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Tab: One\nCategory: A\nTab: Two\nCategory: B\n")
	req := httptest.NewRequest("GET", "/deleteTab?index=1&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := DeleteTabHandler(w, req); err != ErrHandled {
		t.Fatalf("DeleteTabHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	tabs := PreprocessBookmarks(got)
	if len(tabs) != 1 || tabs[0].Pages[0].Blocks[0].Columns[0].Categories[0].Name != "A" {
		t.Fatalf("tab not deleted correctly: %#v", tabs)
	}
}

func TestMoveTabHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Tab: One\nCategory: A\nTab: Two\nCategory: B\n")
	req := httptest.NewRequest("GET", "/moveTab?index=1&dir=up&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := MoveTabHandler(w, req); err != ErrHandled {
		t.Fatalf("MoveTabHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	tabs := PreprocessBookmarks(got)
	if tabs[0].Name != "Two" {
		t.Fatalf("tab not moved: %#v", tabs[0].Name)
	}
}

func TestDeletePageHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Page: A\nCategory: X\nPage: B\nCategory: Y\n")
	req := httptest.NewRequest("GET", "/deletePage?tab=0&index=1&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := DeletePageHandler(w, req); err != ErrHandled {
		t.Fatalf("DeletePageHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	pages := PreprocessBookmarks(got)[0].Pages
	if len(pages) != 1 || pages[0].Name != "A" {
		t.Fatalf("page not deleted correctly: %#v", pages)
	}
}

func TestMovePageHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Page: A\nCategory: X\nPage: B\nCategory: Y\n")
	req := httptest.NewRequest("GET", "/movePage?tab=0&index=1&dir=up&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := MovePageHandler(w, req); err != ErrHandled {
		t.Fatalf("MovePageHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	pages := PreprocessBookmarks(got)[0].Pages
	if pages[0].Name != "B" {
		t.Fatalf("page not moved: %#v", pages[0].Name)
	}
}

func TestDeleteCategoryHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\nCategory: B\n")
	req := httptest.NewRequest("GET", "/deleteCategory?tab=0&page=0&block=0&col=0&index=1&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := DeleteCategoryHandler(w, req); err != ErrHandled {
		t.Fatalf("DeleteCategoryHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	cats := PreprocessBookmarks(got)[0].Pages[0].Blocks[0].Columns[0].Categories
	if len(cats) != 1 || cats[0].Name != "A" {
		t.Fatalf("category not deleted: %#v", cats)
	}
}

func TestMoveCategoryHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\nCategory: B\n")
	req := httptest.NewRequest("GET", "/moveCategory?tab=0&page=0&block=0&col=0&index=1&dir=up&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := MoveCategoryHandler(w, req); err != ErrHandled {
		t.Fatalf("MoveCategoryHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	cats := PreprocessBookmarks(got)[0].Pages[0].Blocks[0].Columns[0].Categories
	if cats[0].Name != "B" {
		t.Fatalf("category not moved: %#v", cats[0].Name)
	}
}

func TestDeleteEntryHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\nhttps://a A\n")
	req := httptest.NewRequest("GET", "/deleteEntry?cat=0&index=0&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := DeleteEntryHandler(w, req); err != ErrHandled {
		t.Fatalf("DeleteEntryHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if strings.Contains(got, "https://a A") {
		t.Fatalf("entry not deleted")
	}
}

func TestMoveEntryHandler(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\nhttps://a A\nhttps://b B\n")
	req := httptest.NewRequest("GET", "/moveEntry?cat=0&index=1&dir=up&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := MoveEntryHandler(w, req); err != ErrHandled {
		t.Fatalf("MoveEntryHandler: %v", err)
	}
	got, _, _ := p.GetBookmarks(context.Background(), user, "refs/heads/main", nil)
	if !strings.Contains(got, "https://b B\nhttps://a A") {
		t.Fatalf("entry not moved: %q", got)
	}
}
