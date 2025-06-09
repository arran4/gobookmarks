package gobookmarks

import "time"

var (
	UseCssColumns bool
	Namespace     string
	SiteTitle     string

	GithubServer string
	GitlabServer string

	GithubClientID     string
	GithubClientSecret string
	GitlabClientID     string
	GitlabClientSecret string

	OauthRedirectURL string
	FaviconCacheDir  string
	FaviconCacheSize int64
)

const (
	DefaultFaviconCacheSize   int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconCacheMaxAge time.Duration = 24 * time.Hour
)