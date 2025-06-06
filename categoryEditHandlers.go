package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func EditCategoryPage(w http.ResponseWriter, r *http.Request) error {
	idxStr := r.URL.Query().Get("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	login, _ := session.Values["UserLogin"].(string)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")

	bookmarks, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	text, err := ExtractCategoryByIndex(bookmarks, idx)
	if err != nil {
		return fmt.Errorf("ExtractCategory: %w", err)
	}
	data := struct {
		*CoreData
		Error string
		Index int
		Text  string
		Sha   string
	}{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Error:    r.URL.Query().Get("error"),
		Index:    idx,
		Text:     text,
		Sha:      sha,
	}
	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "editCategory.gohtml", data); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	return nil
}
