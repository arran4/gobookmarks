package main

import (
	"database/sql"
	"errors"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
)

func bookmarksEditPage(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		*CoreData
		BookmarkContent string
		Bid             interface{}
	}

	data := Data{
		CoreData:        r.Context().Value(ContextValues("coreData")).(*CoreData),
		BookmarkContent: "Category: Example 1\nhttp://www.google.com.au Google\nColumn\nCategory: Example 2\nhttp://www.google.com.au Google\nhttp://www.google.com.au Google\n",
	}
	queries := r.Context().Value(ContextValues("queries")).(*Queries)
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	userRef, _ := session.Values["UserRef"].(string)

	bookmarks, err := queries.GetBookmarksForUser(r.Context(), userRef)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
		default:
			log.Printf("error getBookmarksForUser: %s", err)
			http.Error(w, "ERROR", 500)
			return
		}
	} else {
		data.BookmarkContent = bookmarks.List.String
		data.Bid = bookmarks.Idbookmarks
	}

	if err := getCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "bookmarksEditPage.gohtml", data); err != nil {
		log.Printf("Template Error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func bookmarksEditSaveActionPage(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	queries := r.Context().Value(ContextValues("queries")).(*Queries)
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	userRef, _ := session.Values["UserRef"].(string)

	if err := queries.UpdateBookmarks(r.Context(), UpdateBookmarksParams{
		List: sql.NullString{
			String: text,
			Valid:  true,
		},
		Userreference: userRef,
	}); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}

	http.Redirect(w, r, "/bookmarks/mine", http.StatusTemporaryRedirect)

}

func bookmarksEditCreateActionPage(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	queries := r.Context().Value(ContextValues("queries")).(*Queries)
	session := r.Context().Value(ContextValues("session")).(*sessions.Session)
	userRef, _ := session.Values["UserRef"].(string)

	if err := queries.CreateBookmarks(r.Context(), CreateBookmarksParams{
		List: sql.NullString{
			String: text,
			Valid:  true,
		},
		Userreference: userRef,
	}); err != nil {
		http.Redirect(w, r, "?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}

	http.Redirect(w, r, "/bookmarks/mine", http.StatusTemporaryRedirect)

}
