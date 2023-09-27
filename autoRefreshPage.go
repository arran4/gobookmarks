package gobookmarks

import (
	"fmt"
	"net/http"
)

func TaskDoneAutoRefreshPage(w http.ResponseWriter, r *http.Request) error {
	type Data struct {
		*CoreData
		Error string
	}

	data := Data{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Error:    r.URL.Query().Get("error"),
	}

	data.AutoRefresh = r.URL.Query().Get("error") == ""

	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "taskDoneAutoRefreshPage.gohtml", data); err != nil {
		return fmt.Errorf("template Error: %w", err)
	}
	return nil
}

func taskRedirectWithoutQueryArgs(w http.ResponseWriter, r *http.Request) {
	u := r.URL
	u.RawQuery = ""
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}
