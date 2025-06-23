package gobookmarks

import (
	"net/http"
)

// StartEditMode enables edit mode by adding an "edit=1" query parameter.
func StartEditMode(w http.ResponseWriter, r *http.Request) error {
	qs := r.URL.Query()
	qs.Set("edit", "1")
	r.URL.RawQuery = qs.Encode()
	return nil
}

// StopEditMode disables edit mode by removing the "edit" query parameter.
func StopEditMode(w http.ResponseWriter, r *http.Request) error {
	qs := r.URL.Query()
	qs.Del("edit")
	r.URL.RawQuery = qs.Encode()
	return nil
}
