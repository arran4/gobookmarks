package gobookmarks

import "time"

const (
	DefaultFaviconCacheSize     int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconMaxCacheCount int           = 1000
	DefaultFaviconCacheMaxAge   time.Duration = 24 * time.Hour
	DefaultCommitsPerPage       int           = 100
)
