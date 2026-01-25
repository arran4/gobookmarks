package gobookmarks

import "strings"

// GetRepoName returns the repository name based on the current
// configuration and build mode. When running a development build the name is
// suffixed with "-dev". The Namespace value is appended if supplied.
func (c *Configuration) GetRepoName() string {
	ns := c.Namespace
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
