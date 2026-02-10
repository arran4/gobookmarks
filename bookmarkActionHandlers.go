package gobookmarks

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

func BookmarksEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")
	cfg := r.Context().Value(ContextValues("configuration")).(*Configuration)
	repoName := cfg.GetRepoName()

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	_, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		if errors.Is(err, ErrSignedOut) {
			return ErrSignedOut
		}
		if errors.Is(err, ErrRepoNotFound) {
			if p := providerFromContext(r.Context()); p != nil {
				if err := p.CreateRepo(r.Context(), login, token, repoName); err == nil {
					if err := CreateBookmarks(r.Context(), login, token, branch, text); err == nil {
						http.Redirect(w, r, "/edit?ref=refs/heads/"+branch, http.StatusSeeOther)
						return ErrHandled
					}
				}
			}
			return fmt.Errorf("repository not found")
		}
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	if sha != "" && curSha != sha {
		return fmt.Errorf("bookmark modified concurrently")
	}

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, text, curSha); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return fmt.Errorf("repository not found")
		}
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}

func BookmarksEditCreateAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	if err := CreateBookmarks(r.Context(), login, token, branch, text); err != nil {
		return fmt.Errorf("crateBookmark error: %w", err)
	}
	return nil
}

func CategoryEditSaveAction(w http.ResponseWriter, r *http.Request) error {
	text := r.PostFormValue("text")
	idxStr := r.URL.Query().Get("index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	githubUser, _ := session.Values["GithubUser"].(*User)
	token, _ := session.Values["Token"].(*oauth2.Token)
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")
	sha := r.PostFormValue("sha")

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
	updated, err := ReplaceCategoryByIndex(currentBookmarks, idx, text)
	if err != nil {
		return fmt.Errorf("ReplaceCategory: %w", err)
	}

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}

func CategoryMoveBeforeAction(w http.ResponseWriter, r *http.Request) error {
	fromStr := r.PostFormValue("from")
	toStr := r.PostFormValue("to")
	pageSha := r.PostFormValue("pageSha")
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")

	fromIdx, err := strconv.Atoi(fromStr)
	if err != nil {
		return fmt.Errorf("invalid from index: %w", err)
	}
	toIdx, err := strconv.Atoi(toStr)
	if err != nil {
		return fmt.Errorf("invalid to index: %w", err)
	}

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

	tabs := ParseBookmarks(currentBookmarks)

	page := PageForCategory(tabs, fromIdx)
	if page == nil {
		return fmt.Errorf("category index %d not found", fromIdx)
	}
	if pageSha != "" && page.Sha() != pageSha {
		return fmt.Errorf("bookmark page modified concurrently")
	}

	if err := tabs.MoveCategoryBefore(fromIdx, toIdx); err != nil {
		return fmt.Errorf("MoveCategory: %w", err)
	}
	updated := tabs.String()

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}

func CategoryMoveEndAction(w http.ResponseWriter, r *http.Request) error {
	fromStr := r.PostFormValue("from")
	pageSha := r.PostFormValue("pageSha")
	destPageSha := r.PostFormValue("destPageSha")
	destColStr := r.PostFormValue("destCol")
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")

	fromIdx, err := strconv.Atoi(fromStr)
	if err != nil {
		return fmt.Errorf("invalid from index: %w", err)
	}
	destCol, err := strconv.Atoi(destColStr)
	if destColStr == "" || err != nil {
		destCol = -1
	}

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

	tabs := ParseBookmarks(currentBookmarks)

	page := PageForCategory(tabs, fromIdx)
	if page == nil {
		return fmt.Errorf("category index %d not found", fromIdx)
	}
	if pageSha != "" && page.Sha() != pageSha {
		return fmt.Errorf("bookmark page modified concurrently")
	}

	destPage := FindPageBySha(tabs, destPageSha)
	if destPage == nil {
		destPage = tabs[len(tabs)-1].Pages[len(tabs[len(tabs)-1].Pages)-1]
	}

	if destCol < 0 {
		lastBlock := destPage.Blocks[len(destPage.Blocks)-1]
		destCol = len(lastBlock.Columns) - 1
	}

	if err := tabs.MoveCategoryToEnd(fromIdx, destPage, destCol); err != nil {
		return fmt.Errorf("MoveCategory: %w", err)
	}
	updated := tabs.String()

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}

func CategoryMoveNewColumnAction(w http.ResponseWriter, r *http.Request) error {
	fromStr := r.PostFormValue("from")
	pageSha := r.PostFormValue("pageSha")
	destPageSha := r.PostFormValue("destPageSha")
	destColStr := r.PostFormValue("destCol")
	branch := r.PostFormValue("branch")
	ref := r.PostFormValue("ref")

	fromIdx, err := strconv.Atoi(fromStr)
	if err != nil {
		return fmt.Errorf("invalid from index: %w", err)
	}
	destCol, _ := strconv.Atoi(destColStr)

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

	tabs := ParseBookmarks(currentBookmarks)

	page := PageForCategory(tabs, fromIdx)
	if page == nil {
		return fmt.Errorf("category index %d not found", fromIdx)
	}
	if pageSha != "" && page.Sha() != pageSha {
		return fmt.Errorf("bookmark page modified concurrently")
	}

	destPage := FindPageBySha(tabs, destPageSha)
	if destPage == nil {
		destPage = tabs[len(tabs)-1].Pages[len(tabs[len(tabs)-1].Pages)-1]
	}

	if err := tabs.MoveCategoryNewColumn(fromIdx, destPage, destCol); err != nil {
		return fmt.Errorf("MoveCategory: %w", err)
	}
	updated := tabs.String()

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, updated, curSha); err != nil {
		return fmt.Errorf("updateBookmark error: %w", err)
	}
	return nil
}
