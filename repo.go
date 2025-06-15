package gobookmarks

import "strings"

var RepoName = GetBookmarksRepoName()

// GetBookmarksRepoName returns the repository name based on the current
// configuration and build mode. When running a development build the name is
// suffixed with "-dev". The Namespace value is appended if supplied.
func GetBookmarksRepoName() string {
	ns := Namespace
	if strings.EqualFold(version, "dev") {
		if ns == "" {
			ns = version
		}
	}

	name := "MyBookmarks"
	if ns != "" {
		name += "-" + ns
	}
	return name
}
