package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gb "github.com/arran4/gobookmarks"
	"github.com/gorilla/sessions"
)

func TestRunHandlerChain_UserErrorRedirect(t *testing.T) {
	gb.SessionName = "testsess"
	gb.SessionStore = sessions.NewCookieStore([]byte("secret"))

	req := httptest.NewRequest("GET", "/submit", nil)
	req.Header.Set("Referer", "/form")
	ctx := context.WithValue(req.Context(), gb.ContextValues("coreData"), &gb.CoreData{})
	req = req.WithContext(ctx)

	h := runHandlerChain(func(w http.ResponseWriter, r *http.Request) error {
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
