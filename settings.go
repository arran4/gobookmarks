package gobookmarks

import "time"

var (
	UseCssColumns bool
	Namespace     string
	SiteTitle     string
	NoFooter      bool

	GithubServer string
	GitlabServer string

	GithubClientID     string
	GithubClientSecret string
	GitlabClientID     string
	GitlabClientSecret string

	OauthRedirectURL string
	FaviconCacheDir  string
	FaviconCacheSize int64

	LocalGitPath string

	DBConnectionProvider string
	DBConnectionString   string

	CommitsPerPage int
)

const (
	DefaultFaviconCacheSize   int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconCacheMaxAge time.Duration = 24 * time.Hour
	DefaultCommitsPerPage     int           = 100
)
