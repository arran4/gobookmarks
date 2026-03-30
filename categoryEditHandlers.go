package gobookmarks

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
	"strings"
)

func EditCategoryPage(w http.ResponseWriter, r *http.Request) error {
	idxStr := r.URL.Query().Get("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
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
	text, err := ExtractCategoryByIndex(bookmarks, idx)
	if err != nil {
		return fmt.Errorf("ExtractCategory: %w", err)
	}

	col, _ := strconv.Atoi(r.URL.Query().Get("col"))

	data := struct {
		*CoreData `json:"-"`
		Error     string `json:"error"`
		Index     int    `json:"index"`
		Text      string `json:"text"`
		Sha       string `json:"sha"`
		Col       int    `json:"col"`
	}{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Error:    r.URL.Query().Get("error"),
		Index:    idx,
		Text:     text,
		Sha:      sha,
		Col:      col,
	}

	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(data)
	}

	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "editCategory.gohtml", data); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	return nil
}
