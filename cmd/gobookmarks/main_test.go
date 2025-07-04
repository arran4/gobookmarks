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
	"golang.org/x/oauth2"
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

func TestRunTemplate_BufferedError(t *testing.T) {
	gb.SessionName = "testsess"
	gb.SessionStore = sessions.NewCookieStore([]byte("secret"))
	gb.DBConnectionProvider = ""

	req := httptest.NewRequest("GET", "/", nil)
	sess, _ := gb.SessionStore.New(req, gb.SessionName)
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
