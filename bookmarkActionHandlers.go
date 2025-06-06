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

	login := ""
	if githubUser != nil {
		login = githubUser.Login
	}

	_, curSha, err := GetBookmarks(r.Context(), login, ref, token)
	if err != nil {
		return fmt.Errorf("GetBookmarks: %w", err)
	}
	if sha != "" && curSha != sha {
		return fmt.Errorf("bookmark modified concurrently")
	}

	repoName := RepoName
	if r.PostFormValue("repoName") != "" {
		repoName = r.PostFormValue("repoName")
	}
	if r.PostFormValue("createRepo") == "1" {
		if err := ActiveProvider.CreateRepo(r.Context(), login, token, repoName); err != nil {
			return renderCreateRepoPrompt(w, r, repoName, text, branch, ref, sha, err)
		}
		RepoName = repoName
		if err := CreateBookmarks(r.Context(), login, token, branch, text); err != nil {
			return fmt.Errorf("createBookmark error: %w", err)
		}
		return nil
	}

	if err := UpdateBookmarks(r.Context(), login, token, ref, branch, text, curSha); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return renderCreateRepoPrompt(w, r, repoName, text, branch, ref, sha, nil)
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

func renderCreateRepoPrompt(w http.ResponseWriter, r *http.Request, repoName, text, branch, ref, sha string, err error) error {
	data := struct {
		*CoreData
		RepoName string
		Text     string
		Branch   string
		Ref      string
		Sha      string
		Error    string
	}{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		RepoName: repoName,
		Text:     text,
		Branch:   branch,
		Ref:      ref,
		Sha:      sha,
	}
	if err != nil {
		data.Error = err.Error()
	}
	if tplErr := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "createRepo.gohtml", data); tplErr != nil {
		return fmt.Errorf("template: %w", tplErr)
	}
	return ErrHandled
}
