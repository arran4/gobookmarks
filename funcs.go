package main

import (
	"html/template"
	"net/http"
	"strings"
	"time"
)

func NewFuncs(r *http.Request) template.FuncMap {
	return map[string]any{
		"now": func() time.Time { return time.Now() },
		"firstline": func(s string) string {
			return strings.Split(s, "\n")[0]
		},
		"left": func(i int, s string) string {
			l := len(s)
			if l > i {
				l = i
			}
			return s[:l]
		},
		"OAuth2URL": func() string {
			return oauth2Config.AuthCodeURL("")
		},
	}
}
