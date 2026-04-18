package gobookmarks

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
)

// TabFromRequest extracts the tab index from either the path parameters or the query string.
func TabFromRequest(r *http.Request) int {
	if r == nil {
		return 0
	}
	if vars := mux.Vars(r); vars != nil {
		if tabStr, ok := vars["tab"]; ok {
			if tabIdx, err := strconv.Atoi(tabStr); err == nil {
				return tabIdx
			}
		}
	}
	if tabS := r.URL.Query().Get("tab"); tabS != "" {
		if tabI, err := strconv.Atoi(tabS); err == nil {
			return tabI
		}
	}
	if tabS := r.PostFormValue("tab"); tabS != "" {
		if tabI, err := strconv.Atoi(tabS); err == nil {
			return tabI
		}
	}
	return 0
}

// TabPath returns the semantic path for a tab index (0 is the root tab).
func TabPath(tab int) string {
	if tab <= 0 {
		return "/"
	}
	return fmt.Sprintf("/tab/%d", tab)
}

// TabEditPath returns the edit endpoint for a tab index.
func TabEditPath(tab int) string {
	if tab <= 0 {
		return "/editTab"
	}
	return fmt.Sprintf("/tab/%d/edit", tab)
}

// TabHref builds the link to a tab, preserving ref when provided.
func TabHref(tab int, ref string) string {
	path := TabPath(tab)
	if ref == "" {
		return path
	}
	return fmt.Sprintf("%s?ref=%s", path, url.QueryEscape(ref))
}

// TabEditHref builds the link to edit a tab at the semantic path.
func TabEditHref(tab int, ref, name string) string {
	path := TabEditPath(tab)
	params := []string{}
	if ref != "" {
		params = append(params, "ref", ref)
	}
	if name != "" {
		params = append(params, "name", name)
	}
	return AppendQueryParams(path, params...)
}

// AppendQueryParams appends key/value query params to the provided URL string.
func AppendQueryParams(rawURL string, params ...string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	for i := 0; i+1 < len(params); i += 2 {
		q.Set(params[i], params[i+1])
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// PageFragmentFromIndex converts a zero-based page index string to a 1-based fragment identifier.
func PageFragmentFromIndex(pageStr string) string {
	if pageStr == "" {
		return ""
	}
	if pageIdx, err := strconv.Atoi(pageStr); err == nil {
		return fmt.Sprintf("page%d", pageIdx+1)
	}
	return "page" + pageStr
}
