package gobookmarks

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/arran4/gobookmarks/core"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func AddCategoryPage(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	session := GetCore(r.Context()).GetSession()
	githubUser, _ := session.Values["GithubUser"].(*core.BasicUser)
	token, _ := session.Values["Token"].(*oauth2.Token)
	ref := r.URL.Query().Get("ref")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	_, sha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}

	col, _ := strconv.Atoi(r.URL.Query().Get("col"))

	data := struct {
		*core.CoreData
		Error string
		Index int
		Text  string
		Sha   string
		Col   int
	}{
		CoreData: r.Context().Value(core.ContextValues("coreData")).(*core.CoreData),
		Error:    r.URL.Query().Get("error"),
		Index:    -1,
		Text:     "Category: ",
		Sha:      sha,
		Col:      col,
	}

	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "editCategory.gohtml", data); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	return nil
}

func CategoryAddSaveAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")
	tabIdx, _ := strconv.Atoi(r.PostFormValue("tab"))
	pageIdx, _ := strconv.Atoi(r.PostFormValue("page"))
	colIdx, _ := strconv.Atoi(r.PostFormValue("col"))

	session := r.Context().Value(core.ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*core.BasicUser)
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
	if pageIdx < 0 || pageIdx >= len(list[tabIdx].Pages) {
		pageIdx = len(list[tabIdx].Pages) - 1
	}
	page := list[tabIdx].Pages[pageIdx]
	lastBlock := page.Blocks[len(page.Blocks)-1]
	if colIdx < 0 || colIdx >= len(lastBlock.Columns) {
		colIdx = len(lastBlock.Columns) - 1
	}
	column := lastBlock.Columns[colIdx]

	parsed := ParseBookmarks(text)
	if len(parsed) == 0 || len(parsed[0].Pages) == 0 || len(parsed[0].Pages[0].Blocks) == 0 || len(parsed[0].Pages[0].Blocks[0].Columns) == 0 || len(parsed[0].Pages[0].Blocks[0].Columns[0].Categories) == 0 {
		return fmt.Errorf("invalid category text")
	}
	cat := parsed[0].Pages[0].Blocks[0].Columns[0].Categories[0]
	column.AddCategory(cat)

	updated := list.String()

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}
