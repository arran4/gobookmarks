package app

// Config holds the application configuration.
type Config struct {
	SessionName          string
	BaseURL              string
	ExternalURL          string
	DevMode              bool
	GithubClientID       string
	GithubSecret         string
	GithubServer         string
	GitlabClientID       string
	GitlabSecret         string
	GitlabServer         string
	Title                string
	CssColumns           bool
	NoFooter             bool
	LocalGitPath         string
	CommitsPerPage       int
	FaviconCacheDir      string
	FaviconCacheSize     int64
	FaviconMaxCacheCount int
}
