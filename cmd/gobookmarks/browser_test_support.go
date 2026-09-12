package main

import (
	"golang.org/x/net/publicsuffix"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type Browser struct {
	Handler http.Handler
	Jar     *cookiejar.Jar
	Origin  *url.URL
}

func NewBrowser(handler http.Handler, origin string) (*Browser, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	return &Browser{
		Handler: handler,
		Jar:     jar,
		Origin:  u,
	}, nil
}

type TestResponse struct {
	Response *http.Response
	Cookies  []*http.Cookie
}

func (b *Browser) Do(method, target string, body io.Reader) (*TestResponse, error) {
	return b.DoWithHeaders(method, target, body, nil)
}

func (b *Browser) DoForm(target string, values url.Values) (*TestResponse, error) {
	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/x-www-form-urlencoded")
	return b.DoWithHeaders("POST", target, strings.NewReader(values.Encode()), hdr)
}

func (b *Browser) DoWithHeaders(method, target string, body io.Reader, headers http.Header) (*TestResponse, error) {
	reqURL, err := b.Origin.Parse(target)
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest(method, reqURL.String(), body)
	for k, vv := range headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	for _, cookie := range b.Jar.Cookies(reqURL) {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	b.Handler.ServeHTTP(rec, req)

	resp := rec.Result()

	cookies := resp.Cookies()
	if len(cookies) > 0 {
		b.Jar.SetCookies(reqURL, cookies)
	}

	return &TestResponse{
		Response: resp,
		Cookies:  cookies,
	}, nil
}

type CookieIdentity struct {
	Name   string
	Domain string
	Path   string
}

type CookieMetadata struct {
	Identity CookieIdentity
	MaxAge   int
	Expires  time.Time
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
	IsLive   bool
	IsExpiry bool
}

// AssertOrderedSetCookieMutations verifies that across the response's Set-Cookie sequence:
// 1. For each cookie identity (Name, Domain, Path), a live authenticated cookie is NEVER subsequently followed by an expiry/deletion cookie.
// 2. Failure messages only contain non-sensitive cookie metadata, never cookie values or tokens.
func AssertOrderedSetCookieMutations(t *testing.T, cookies []*http.Cookie) map[CookieIdentity][]CookieMetadata {
	t.Helper()
	mutationsByIdentity := make(map[CookieIdentity][]CookieMetadata)

	for _, c := range cookies {
		isExpiry := c.MaxAge < 0 || (!c.Expires.IsZero() && c.Expires.Before(time.Now()))
		isLive := !isExpiry
		id := CookieIdentity{Name: c.Name, Domain: c.Domain, Path: c.Path}
		meta := CookieMetadata{
			Identity: id,
			MaxAge:   c.MaxAge,
			Expires:  c.Expires,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
			SameSite: c.SameSite,
			IsLive:   isLive,
			IsExpiry: isExpiry,
		}
		mutationsByIdentity[id] = append(mutationsByIdentity[id], meta)
	}

	for id, mutations := range mutationsByIdentity {
		seenLive := false
		for _, m := range mutations {
			if m.IsLive {
				seenLive = true
			} else if m.IsExpiry && seenLive {
				t.Fatalf("Ordered Set-Cookie regression: live authenticated cookie was subsequently followed by an expiry/deletion cookie for identity (Name=%s, Domain=%s, Path=%s): mutation_history=%+v",
					id.Name, id.Domain, id.Path, mutations)
			}
		}
	}

	return mutationsByIdentity
}
