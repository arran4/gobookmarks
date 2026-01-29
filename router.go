package gobookmarks

import (
	"database/sql"
	"net/http"

	"github.com/arran4/gorillamuxlogic"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

// RouterConfig holds the dependencies and configuration for the gobookmarks application
type RouterConfig struct {
	// DB is the database connection
	DB *sql.DB

	// UserProvider handles user authentication and lookup
	UserProvider UserProvider

	// SessionStore handles session management
	SessionStore sessions.Store
	// SessionName is the name of the session cookie
	SessionName string

	// BaseURL is the prefix for all routes (e.g. "/bookmarks")
	BaseURL string

	// ExternalURL is the public facing URL
	ExternalURL string

	// DevMode enables development features
	DevMode bool
}

// UserProvider defines the interface for external user management
type UserProvider interface {
	// CurrentUser returns the current user from the request context
	CurrentUser(r *http.Request) (*User, error)
	// IsLoggedIn checks if a user is logged in
	IsLoggedIn(r *http.Request) bool
}

// NewRouter creates a new router with the given configuration
func NewRouter(cfg *RouterConfig) http.Handler {
	// Initialize globals temporarily until full refactor
	if cfg.ExternalURL != "" {
		// ExternalUrl is a global in gobookmarks package, need to verify
	}
	// Note: We are not handling all globals yet, focusing on router structure first

	r := mux.NewRouter()

	r.Use(UserAdderMiddleware) // Middleware needs to be adapted to use UserProvider if present
	r.Use(CoreAdderMiddleware)

	r.HandleFunc("/main.css", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(GetMainCSSData())
	}).Methods("GET")
	r.HandleFunc("/favicon.ico", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(GetFavicon())
	}).Methods("GET")

	// Development helpers to toggle layout mode
	if cfg.DevMode {
		r.HandleFunc("/_css", runHandlerChain(EnableCssColumnsAction, redirectToHandler("/"))).Methods("GET")
		r.HandleFunc("/_table", runHandlerChain(DisableCssColumnsAction, redirectToHandler("/"))).Methods("GET")
	}

	// News
	r.Handle("/", http.HandlerFunc(runTemplate("mainPage.gohtml"))).Methods("GET")
	r.Handle("/tab", http.HandlerFunc(runTemplate("mainPage.gohtml"))).Methods("GET")
	r.Handle("/tab/{tab}", http.HandlerFunc(runTemplate("mainPage.gohtml"))).Methods("GET")
	r.HandleFunc("/", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

	r.HandleFunc("/edit", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/edit", runTemplate("edit.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/edit", runTemplate("edit.gohtml")).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(HasError())
	r.HandleFunc("/edit", runHandlerChain(BookmarksEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSave))
	r.HandleFunc("/edit", runHandlerChain(BookmarksEditSaveAction, TaskDoneAutoRefreshPage)).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndDone))
	r.HandleFunc("/edit", runHandlerChain(BookmarksEditSaveAction, StopEditMode, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndStopEditing))
	r.HandleFunc("/edit", runHandlerChain(BookmarksEditCreateAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Create"))
	r.HandleFunc("/edit", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

	r.HandleFunc("/startEditMode", runHandlerChain(StartEditMode, redirectToHandlerTabPage("/"))).Methods("POST", "GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/stopEditMode", runHandlerChain(StopEditMode, redirectToHandlerTabPage("/"))).Methods("POST", "GET").MatcherFunc(RequiresAnAccount())

	r.HandleFunc("/editCategory", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/editCategory", runHandlerChain(EditCategoryPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/editCategory", runHandlerChain(CategoryEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSave))
	r.HandleFunc("/editCategory", runHandlerChain(CategoryEditSaveAction, TaskDoneAutoRefreshPage)).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndDone))
	r.HandleFunc("/editCategory", runHandlerChain(CategoryEditSaveAction, StopEditMode, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndStopEditing))
	r.HandleFunc("/editCategory", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")
	r.HandleFunc("/addCategory", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/addCategory", runHandlerChain(AddCategoryPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/addCategory", runHandlerChain(CategoryAddSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSave))
	r.HandleFunc("/addCategory", runHandlerChain(CategoryAddSaveAction, TaskDoneAutoRefreshPage)).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndDone))
	r.HandleFunc("/addCategory", runHandlerChain(CategoryAddSaveAction, StopEditMode, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndStopEditing))
	r.HandleFunc("/addCategory", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")
	r.HandleFunc("/moveCategory", runHandlerChain(CategoryMoveBeforeAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/moveCategoryEnd", runHandlerChain(CategoryMoveEndAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/moveCategoryNewColumn", runHandlerChain(CategoryMoveNewColumnAction)).Methods("POST").MatcherFunc(RequiresAnAccount())

	r.HandleFunc("/editTab", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/editTab", runHandlerChain(EditTabPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/editTab", runHandlerChain(TabEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSave))
	r.HandleFunc("/editTab", runHandlerChain(TabEditSaveAction, TaskDoneAutoRefreshPage)).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndDone))
	r.HandleFunc("/editTab", runHandlerChain(TabEditSaveAction, StopEditMode, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndStopEditing))
	r.HandleFunc("/editTab", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")
	r.HandleFunc("/tab/{tab}/edit", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/tab/{tab}/edit", runHandlerChain(EditTabPage)).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/tab/{tab}/edit", runHandlerChain(TabEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSave))
	r.HandleFunc("/tab/{tab}/edit", runHandlerChain(TabEditSaveAction, TaskDoneAutoRefreshPage)).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndDone))
	r.HandleFunc("/tab/{tab}/edit", runHandlerChain(TabEditSaveAction, StopEditMode, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndStopEditing))
	r.HandleFunc("/tab/{tab}/edit", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

	r.HandleFunc("/editPage", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/editPage", runHandlerChain(EditPagePage)).Methods("GET").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/editPage", runHandlerChain(PageEditSaveAction, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSave))
	r.HandleFunc("/editPage", runHandlerChain(PageEditSaveAction, TaskDoneAutoRefreshPage)).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndDone))
	r.HandleFunc("/editPage", runHandlerChain(PageEditSaveAction, StopEditMode, redirectToHandlerBranchToRef("/"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher(TaskSaveAndStopEditing))
	r.HandleFunc("/editPage", runHandlerChain(TaskDoneAutoRefreshPage)).Methods("POST")

	r.HandleFunc("/moveTab", runHandlerChain(MoveTabAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/movePage", runHandlerChain(MovePageAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/tab/{tab}/movePage", runHandlerChain(MovePageAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/moveEntry", runHandlerChain(MoveEntryAction)).Methods("POST").MatcherFunc(RequiresAnAccount())
	r.HandleFunc("/tab/{tab}/moveEntry", runHandlerChain(MoveEntryAction)).Methods("POST").MatcherFunc(RequiresAnAccount())

	r.HandleFunc("/history", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/history", runTemplate("history.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())

	r.HandleFunc("/history/commits", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	r.HandleFunc("/status", runTemplate("statusPage.gohtml")).Methods("GET")
	r.HandleFunc("/history/commits", runTemplate("historyCommits.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())

	r.HandleFunc("/login", runTemplate("loginPage.gohtml")).Methods("GET")
	r.HandleFunc("/login/git", runTemplate("gitLoginPage.gohtml")).Methods("GET")
	r.HandleFunc("/login/git", runHandlerChain(GitLoginAction, redirectToHandler("/"))).Methods("POST")
	r.HandleFunc("/signup/git", runHandlerChain(GitSignupAction, redirectToHandler("/login/git"))).Methods("POST")
	r.HandleFunc("/login/sql", runTemplate("sqlLoginPage.gohtml")).Methods("GET")
	r.HandleFunc("/login/sql", runHandlerChain(SqlLoginAction, redirectToHandler("/"))).Methods("POST")
	r.HandleFunc("/signup/sql", runHandlerChain(SqlSignupAction, redirectToHandler("/login/sql"))).Methods("POST")
	r.HandleFunc("/login/{provider}", runHandlerChain(LoginWithProvider)).Methods("GET")
	r.HandleFunc("/logout", runHandlerChain(UserLogoutAction, runTemplate("logoutPage.gohtml"))).Methods("GET")
	r.HandleFunc("/oauth2Callback", runHandlerChain(Oauth2CallbackPage, redirectToHandler("/"))).Methods("GET")

	r.HandleFunc("/proxy/favicon", FaviconProxyHandler).Methods("GET")

	// Create required files if missing
	if !fileExists("cert.pem") || !fileExists("key.pem") {
		CreatePEMFiles()
	}

	return r
}
