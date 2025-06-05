package gobookmarks

import "time"

// RepositoryTag represents a repository tag name
// used for both GitHub and GitLab providers.
type RepositoryTag struct {
	Name string
}

// Branch represents a repository branch name
// used for both GitHub and GitLab providers.
type Branch struct {
	Name string
}

// RepositoryCommit is a simplified commit structure
// that matches the fields used by templates.
type RepositoryCommit struct {
	SHA    string
	Commit struct {
		Message   string
		Committer struct {
			Date  time.Time
			Name  string
			Email string
		}
	}
}
