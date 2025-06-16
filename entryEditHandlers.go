package gobookmarks

import (
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

type entryData struct {
	*CoreData
	Error  string
	Cat    int
	Entry  int
	Url    string
	Name   string
	Sha    string
	Return string
}

func EditEntryPage(w http.ResponseWriter, r *http.Request) error {
	cStr := r.URL.Query().Get("cat")
	eStr := r.URL.Query().Get("entry")
	cIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid category index: %w", err)
	}
	eIdx, err := strconv.Atoi(eStr)
	if err != nil {
		return fmt.Errorf("invalid entry index: %w", err)
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
	tabs := PreprocessBookmarks(bookmarks)
	var entry *BookmarkEntry
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for _, col := range b.Columns {
					for _, cat := range col.Categories {
						if cat.Index == cIdx {
							if eIdx >= 0 && eIdx < len(cat.Entries) {
								entry = cat.Entries[eIdx]
							}
							break
						}
					}
				}
			}
		}
	}
	if entry == nil {
		return fmt.Errorf("entry not found")
	}
	data := entryData{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Error:    r.URL.Query().Get("error"),
		Cat:      cIdx,
		Entry:    eIdx,
		Url:      entry.Url,
		Name:     entry.Name,
		Sha:      sha,
		Return:   r.URL.Query().Get("return"),
	}
	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "editEntry.gohtml", data); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	return nil
}

func EntryEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	cStr := r.URL.Query().Get("cat")
	eStr := r.URL.Query().Get("entry")
	cIdx, err := strconv.Atoi(cStr)
	if err != nil {
		return fmt.Errorf("invalid category index: %w", err)
	}
	eIdx, err := strconv.Atoi(eStr)
	if err != nil {
		return fmt.Errorf("invalid entry index: %w", err)
	}
	url := r.PostFormValue("url")
	name := r.PostFormValue("name")
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

	current, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	if sha != "" && curSha != sha {
		return fmt.Errorf("bookmark modified concurrently")
	}
	tabs := PreprocessBookmarks(current)
	var cat *BookmarkCategory
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for _, col := range b.Columns {
					for _, c := range col.Categories {
						if c.Index == cIdx {
							cat = c
							break
						}
					}
				}
			}
		}
	}
	if cat == nil {
		return fmt.Errorf("category not found")
	}
	if eIdx < 0 || eIdx >= len(cat.Entries) {
		return fmt.Errorf("entry index out of range")
	}
	cat.Entries[eIdx] = &BookmarkEntry{Url: url, Name: name}
	updated := SerializeBookmarks(tabs)
	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}
