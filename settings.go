package gobookmarks

import "time"

var (
	UseCssColumns bool
	DevMode       bool
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

	// ContentSecurityPolicy is the value of the CSP header sent on responses.
	ContentSecurityPolicy string
)

const (
	DefaultFaviconCacheSize   int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconCacheMaxAge time.Duration = 24 * time.Hour
	DefaultCommitsPerPage     int           = 100

	// DefaultContentSecurityPolicy is applied when no policy is configured.
	DefaultContentSecurityPolicy string = "default-src 'self'; img-src 'self' data:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"
)
