package main

import (
	"database/sql"
	"errors"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
	"strings"
)

type BookmarkEntry struct {
	Url  string
	Name string
}

type BookmarkCategory struct {
	Name    string
	Entries []*BookmarkEntry
}

type BookmarkColumn struct {
	Categories []*BookmarkCategory
}

func preprocessBookmarks(bookmarks string) []*BookmarkColumn {
	lines := strings.Split(bookmarks, "\n")
	var result = []*BookmarkColumn{{}}
	var currentCategory *BookmarkCategory

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.EqualFold(line, "column") {
			result = append(result, &BookmarkColumn{})
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) > 0 && strings.EqualFold(parts[0], "Category:") {
			categoryName := strings.Join(parts[1:], " ")
			if currentCategory == nil {
				currentCategory = &BookmarkCategory{Name: categoryName}
			} else if currentCategory.Name != "" {
				result[len(result)-1].Categories = append(result[len(result)-1].Categories, currentCategory)
				currentCategory = &BookmarkCategory{Name: categoryName}
			} else {
				currentCategory.Name = categoryName
			}
		} else if len(parts) > 0 && currentCategory != nil {
			var entry BookmarkEntry
			entry.Url = parts[0]
			entry.Name = parts[0]
			if len(parts) > 1 {
				entry.Name = strings.Join(parts[1:], " ")
			}
			currentCategory.Entries = append(currentCategory.Entries, &entry)
		}
	}

	if currentCategory != nil && currentCategory.Name != "" {
		result[len(result)-1].Categories = append(result[len(result)-1].Categories, currentCategory)
	}

	return result
}

func bookmarksMinePage(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		*CoreData
		Columns []*BookmarkColumn
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
	}

	data := Data{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Columns:  preprocessBookmarks(bookmarks.List.String),
	}

	if err := getCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "bookmarksMinePage.gohtml", data); err != nil {
		log.Printf("Template Error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
