package gobookmarks

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddEntryAndEdit(t *testing.T) {
	p, user, ctx := setupHandlerTest(t, "Category: A\nhttps://a A\n")
	req := httptest.NewRequest("GET", "/addEntry?cat=0&index=1&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := AddEntryHandler(w, req); err != ErrHandled {
		t.Fatalf("AddEntryHandler: %v", err)
	}
	// use redirect location for edit
	loc := w.Result().Header.Get("Location")
	if loc == "" {
		t.Fatalf("no redirect location")
	}
	t.Logf("redirect to %s", loc)
	req2 := httptest.NewRequest("GET", loc, nil)
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()
	err := EditEntryPage(w2, req2)
	if err != nil {
		t.Fatalf("EditEntryPage: %v", err)
	}
	if w2.Code != 200 {
		t.Fatalf("status %d body %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "Name") {
		t.Fatalf("edit page not rendered: %s", w2.Body.String())
	}
	got, _, _ := p.GetBookmarks(ctx, user, "refs/heads/main", nil)
	if !strings.Contains(got, "http://") {
		t.Fatalf("bookmark not inserted: %s", got)
	}
}

func TestAddEntryFormValues(t *testing.T) {
	_, _, ctx := setupHandlerTest(t, "Category: A\n")
	req := httptest.NewRequest("GET", "/addEntry?cat=0&index=0&branch=main&ref=refs/heads/main", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	if err := AddEntryHandler(w, req); err != ErrHandled {
		t.Fatalf("AddEntryHandler: %v", err)
	}
	loc := w.Result().Header.Get("Location")
	req2 := httptest.NewRequest("GET", loc, nil)
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()
	if err := EditEntryPage(w2, req2); err != nil {
		t.Fatalf("EditEntryPage: %v", err)
	}
	body := w2.Body.String()
	if !strings.Contains(body, "value=\"http://\"") {
		t.Fatalf("url field missing placeholder: %s", body)
	}
	if strings.Contains(body, "name\" value=\"http://\"") {
		t.Fatalf("name field incorrectly populated: %s", body)
	}
}
