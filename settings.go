package gobookmarks

import "time"

var (
	UseCssColumns    bool
	Namespace        string
	SiteTitle        string
	GitServer        string
	FaviconCacheDir  string
	FaviconCacheSize int64
)

const (
	DefaultFaviconCacheSize   int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconCacheMaxAge time.Duration = 24 * time.Hour
)
