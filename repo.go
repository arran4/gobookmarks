package gobookmarks

var RepoName = GetBookmarksRepoName()

// GetBookmarksRepoName returns the repository name based on the current
// configuration and build mode. When running a development build the name is
// suffixed with "-dev". The Namespace value is appended if supplied.
func GetBookmarksRepoName() string {
	name := "MyBookmarks"
	if version == "dev" {
		name += "-dev"
	}
	if Namespace != "" {
		name += "-" + Namespace
	}
	return name
}
