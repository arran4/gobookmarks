package a4webbm

import (
	"log"
	"net/http"
)

func TaskDoneAutoRefreshPage(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		*CoreData
		Error string
	}

	data := Data{
		CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		Error:    r.URL.Query().Get("error"),
	}

	data.AutoRefresh = r.URL.Query().Get("error") == ""

	if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, "TaskDoneAutoRefreshPage.gohtml", data); err != nil {
		log.Printf("Template Error: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func taskRedirectWithoutQueryArgs(w http.ResponseWriter, r *http.Request) {
	u := r.URL
	u.RawQuery = ""
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}
