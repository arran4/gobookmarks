package gobookmarks

var RepoName = GetBookmarksRepoName()

func GetBookmarksRepoName() string {
	if Namespace != "" {
		return "MyBookmarks-" + Namespace
	}
	return "MyBookmarks"
}
