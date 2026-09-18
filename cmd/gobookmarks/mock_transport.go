package main

import (
	"bytes"
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

	if strings.Contains(req.URL.Path, "user") {
		body := fmt.Sprintf(`{"login": %q}`, m.userLogin)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     header,
		}, nil
	}

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("{}")),
		Header:     header,
	}, nil
}
