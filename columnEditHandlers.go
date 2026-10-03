package gobookmarks

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func EditColumnGetAction(w http.ResponseWriter, r *http.Request) error {
	tabIdx, err := strconv.Atoi(r.FormValue("tab"))
	if err != nil {
		tabIdx = -1
	}
	pageIdx, err := strconv.Atoi(r.FormValue("page"))
	if err != nil {
		pageIdx = -1
	}
	blockIdx, err := strconv.Atoi(r.FormValue("block"))
	if err != nil {
		blockIdx = -1
	}
	colIdx, err := strconv.Atoi(r.FormValue("col"))
	if err != nil {
		colIdx = -1
	}

	ctx := r.Context()
	v := ctx.Value(ContextValues("coreData"))
	l, ok := v.(*CoreData)
	if !ok {
		return fmt.Errorf("could not get coreData")
	}

	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	bookmarks, actualSha, err := GetBookmarks(ctx, login, r.FormValue("ref"), token)
	if err != nil {
		return err
	}

	tabs := ParseBookmarks(bookmarks)

	if tabIdx < 0 || tabIdx >= len(tabs) {
		return fmt.Errorf("invalid tab index")
	}
	tab := tabs[tabIdx]
	if pageIdx < 0 || pageIdx >= len(tab.Pages) {
		return fmt.Errorf("invalid page index")
	}
	page := tab.Pages[pageIdx]
	if blockIdx < 0 || blockIdx >= len(page.Blocks) {
		return fmt.Errorf("invalid block index")
	}
	block := page.Blocks[blockIdx]
	if colIdx < 0 || colIdx >= len(block.Columns) {
		return fmt.Errorf("invalid column index")
	}
	col := block.Columns[colIdx]

	expectedSha := r.FormValue("sha")
	if expectedSha == "" {
		expectedSha = actualSha
	}

	data := struct {
		*CoreData
		Name  string
		Tab   int
		Page  int
		Block int
		Col   int
		Sha   string
	}{
		CoreData: l,
		Name:     col.Name,
		Tab:      tabIdx,
		Page:     pageIdx,
		Block:    blockIdx,
		Col:      colIdx,
		Sha:      expectedSha,
	}

	tmplName := "editColumn.gohtml"
	if strings.HasSuffix(r.URL.Path, "/modal") {
		tmplName = "_partials/editColumnForm.gohtml"
	}
	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, tmplName, data); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	return nil
}

func EditColumnPostAction(w http.ResponseWriter, r *http.Request) error {
	tabIdx, err := strconv.Atoi(r.FormValue("tab"))
	if err != nil {
		tabIdx = -1
	}
	pageIdx, err := strconv.Atoi(r.FormValue("page"))
	if err != nil {
		pageIdx = -1
	}
	blockIdx, err := strconv.Atoi(r.FormValue("block"))
	if err != nil {
		blockIdx = -1
	}
	colIdx, err := strconv.Atoi(r.FormValue("col"))
	if err != nil {
		colIdx = -1
	}
	newName := strings.TrimSpace(r.FormValue("name"))

	ctx := r.Context()

	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	expectedSha := r.FormValue("sha")

	bookmarks, actualSha, err := GetBookmarks(ctx, login, r.FormValue("ref"), token)
	if err != nil {
		return err
	}

	if expectedSha != actualSha {
		return fmt.Errorf("concurrent modification detected")
	}

	tabs := ParseBookmarks(bookmarks)
	if tabIdx < 0 || tabIdx >= len(tabs) {
		return fmt.Errorf("invalid tab index")
	}
	tab := tabs[tabIdx]
	if pageIdx < 0 || pageIdx >= len(tab.Pages) {
		return fmt.Errorf("invalid page index")
	}
	page := tab.Pages[pageIdx]
	if blockIdx < 0 || blockIdx >= len(page.Blocks) {
		return fmt.Errorf("invalid block index")
	}
	block := page.Blocks[blockIdx]
	if colIdx < 0 || colIdx >= len(block.Columns) {
		return fmt.Errorf("invalid column index")
	}
	col := block.Columns[colIdx]

	col.Name = newName

	newRaw := tabs.String()

	if err := UpdateBookmarks(ctx, login, token, r.FormValue("ref"), r.FormValue("branch"), newRaw, expectedSha); err != nil {
		return err
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return ErrHandled
}
