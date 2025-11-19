package gobookmarks

import (
	"context"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func EditPagePage(w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")
	tabIdx := TabFromRequest(r)
	pageIdx, _ := strconv.Atoi(r.URL.Query().Get("page"))

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	bookmarks, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}

	data := struct {
		*CoreData
		Error string
		Name  string
		Text  string
		Sha   string
	}{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Error:    r.URL.Query().Get("error"),
		Name:     r.URL.Query().Get("name"),
		Text:     "",
		Sha:      sha,
	}

	if pageIdx >= 0 {
		pageText, pageName, err := ExtractPage(bookmarks, tabIdx, pageIdx)
		if err != nil {
			return fmt.Errorf("ExtractPage: %w", err)
		}
		data.Name = pageName
		data.Text = pageText
	}

	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "editPage.gohtml", data); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	return nil
}

func PageEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	name := r.PostFormValue("name")
	text := r.PostFormValue("text")
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")
	tabIdx := TabFromRequest(r)
	pageIdx, pageErr := strconv.Atoi(r.PostFormValue("page"))

	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	currentBookmarks, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	if sha != "" && curSha != sha {
		return fmt.Errorf("bookmark modified concurrently")
	}

	list := ParseBookmarks(currentBookmarks)
	if tabIdx < 0 || tabIdx >= len(list) {
		tabIdx = 0
	}
	parsed := ParseBookmarks("Tab\nPage: " + name + "\n" + text)
	p := parsed[0].Pages[0]
	if pageErr == nil {
		if pageIdx < 0 || pageIdx >= len(list[tabIdx].Pages) {
			return fmt.Errorf("page index out of range")
		}
		list[tabIdx].Pages[pageIdx] = p
	} else {
		newIndex := len(list[tabIdx].Pages)
		list[tabIdx].AddPage(p)
		ctx := context.WithValue(r.Context(), ContextValues("redirectTab"), strconv.Itoa(tabIdx))
		ctx = context.WithValue(ctx, ContextValues("redirectPage"), strconv.Itoa(newIndex))
		*r = *r.WithContext(ctx)
	}

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, list.String(), curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}

	return nil
}
