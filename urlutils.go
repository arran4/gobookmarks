package gobookmarks

import (
	"net/url"
	"strings"
)

// JoinURL joins base and elem ensuring there is exactly one slash between them.
// Additional leading or trailing slashes are removed from elem.
func JoinURL(base, elem string) string {
	base = strings.TrimRight(base, "/")
	elem = strings.Trim(elem, "/")
	if base == "" {
		return "/" + elem
	}
	return base + "/" + elem
}

// IsSafeRedirect reports whether a redirect target is a safe local application path.
// It prevents open redirect vulnerabilities, schema/host hijacking, CRLF injection, and protocol-relative URLs.
func IsSafeRedirect(target string) bool {
	if target == "" || len(target) >= 2048 {
		return false
	}
	// Must start with a single '/'
	if !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") || strings.HasPrefix(target, "/\\") {
		return false
	}
	// Disallow backslashes anywhere in the redirect target
	if strings.Contains(target, "\\") {
		return false
	}
	// Disallow control characters / CRLF
	if strings.ContainsAny(target, "\r\n\t") {
		return false
	}
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	// Disallow scheme and host
	if u.Scheme != "" || u.Host != "" {
		return false
	}
	// Path must be local and not protocol-relative
	if !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return false
	}
	return true
}
