package gobookmarks

import "net/http"

// EnableCssColumnsAction sets the layout to use CSS columns.
func EnableCssColumnsAction(w http.ResponseWriter, r *http.Request) error {
	UseCssColumns = true
	return nil
}

// DisableCssColumnsAction sets the layout to use table columns.
func DisableCssColumnsAction(w http.ResponseWriter, r *http.Request) error {
	UseCssColumns = false
	return nil
}
