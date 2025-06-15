package gobookmarks

import "strings"

// JoinURL joins base and elem ensuring there is exactly one slash between them.
// Additional leading or trailing slashes are removed from elem.
func JoinURL(base, elem string) string {
	base = strings.TrimRight(base, "/")
	elem = strings.TrimLeft(elem, "/")
	if base == "" {
		return "/" + elem
	}
	return base + "/" + elem
}
