package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

func EditTabPage(w http.ResponseWriter, r *http.Request) error {
	tabName := r.URL.Query().Get("name")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	bookmarks, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	text := ""
	if tabName != "" {
		t, err := ExtractTab(bookmarks, tabName)
		if err != nil {
			return fmt.Errorf("ExtractTab: %w", err)
		}
		// drop first line (Tab: ...)
		lines := strings.SplitN(t, "\n", 2)
		if len(lines) == 2 {
			text = lines[1]
		}
	}

	data := struct {
		*CoreData
		Error   string
		Name    string
		OldName string
		Text    string
		Sha     string
	}{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Error:    r.URL.Query().Get("error"),
		Name:     tabName,
		OldName:  tabName,
		Text:     text,
		Sha:      sha,
	}

	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "editTab.gohtml", data); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	return nil
}

func TabEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	oldName := r.URL.Query().Get("name")
	name := r.PostFormValue("name")
	text := r.PostFormValue("text")
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")

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

	var updated string
	if oldName == "" {
		updated = AppendTab(currentBookmarks, name, text)
	} else {
		updated, err = ReplaceTab(currentBookmarks, oldName, name, text)
		if err != nil {
			return fmt.Errorf("ReplaceTab: %w", err)
		}
	}

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}
