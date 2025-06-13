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
	repoName := RepoName

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	_, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			if p := providerFromContext(r.Context()); p != nil {
				if err := p.CreateRepo(r.Context(), login, token, repoName); err == nil {
					if err := CreateBookmarks(r.Context(), login, token, branch, text); err == nil {
						http.Redirect(w, r, "/edit?ref=refs/heads/"+branch, http.StatusTemporaryRedirect)
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
