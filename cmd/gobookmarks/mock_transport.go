package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type mockOAuthRoundTripper struct {
	fakeToken string
	userLogin string
}

func (m *mockOAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")

	if strings.Contains(req.URL.Path, "access_token") {
		body := fmt.Sprintf(`{"access_token": %q, "token_type": "bearer"}`, m.fakeToken)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     header,
		}, nil
	}

	if req.URL.Path == "/user" || strings.HasSuffix(req.URL.Path, "/api/v3/user") {
		body := fmt.Sprintf(`{"login": %q}`, m.userLogin)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     header,
		}, nil
	}
	if strings.Contains(req.URL.Path, "/contents/bookmarks.txt") {
		bookmarks := "Tab: Provider\nPage: Mock\nCategory: Links\nhttps://example.com Mocked provider bookmark"
		body := fmt.Sprintf(`{"content":%q,"encoding":"base64","sha":"scenario-sha"}`, base64.StdEncoding.EncodeToString([]byte(bookmarks)))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(body)), Header: header}, nil
	}
	if strings.Contains(req.URL.Path, "/repos/") {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"name":"gobookmarks"}`)), Header: header}, nil
	}

	return nil, fmt.Errorf("unexpected outbound OAuth/provider request: %s %s", req.Method, req.URL.String())
}
