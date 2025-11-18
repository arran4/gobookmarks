package gobookmarks

import (
	"context"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
	"strings"
)

func EditTabPage(w http.ResponseWriter, r *http.Request) error {
	tabName := r.URL.Query().Get("name")
	tabIdx, _ := strconv.Atoi(r.URL.Query().Get("tab"))
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
	tabs := ParseBookmarks(bookmarks)
	if tabIdx < 0 || tabIdx >= len(tabs) {
		tabIdx = 0
	}
	text := ""
	tabFromQuery := tabName != ""
	if tabName == "" && tabIdx < len(tabs) {
		tabName = tabs[tabIdx].Name
	}
	if tabFromQuery || tabIdx < len(tabs) {
		tabText, err := ExtractTabByIndex(bookmarks, tabIdx)
		if err != nil {
			return fmt.Errorf("ExtractTabByIndex: %w", err)
		}
		lines := strings.SplitN(tabText, "\n", 2)
		hasHeader := len(lines) > 0 && strings.HasPrefix(strings.ToLower(strings.TrimSpace(lines[0])), "tab")
		switch {
		case hasHeader && len(lines) == 2:
			text = lines[1]
		case !hasHeader:
			text = tabText
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
	tabIdx, _ := strconv.Atoi(r.PostFormValue("tab"))
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
	newIndex := len(ParseBookmarks(currentBookmarks))
	if tabIdx >= 0 && tabIdx < len(ParseBookmarks(currentBookmarks)) {
		updated, err = ReplaceTabByIndex(currentBookmarks, tabIdx, name, text)
		if err != nil {
			return fmt.Errorf("ReplaceTabByIndex: %w", err)
		}
	} else if oldName == "" {
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
	if oldName == "" {
		ctx := context.WithValue(r.Context(), ContextValues("redirectTab"), strconv.Itoa(newIndex))
		ctx = context.WithValue(ctx, ContextValues("redirectPage"), "0")
		*r = *r.WithContext(ctx)
	}
	return nil
}
