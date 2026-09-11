package harness_tests

import (
	"golang.org/x/net/publicsuffix"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
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
	reqURL, err := b.Origin.Parse(target)
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest(method, reqURL.String(), body)

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

func TestBrowserFlow(t *testing.T) {
	// This is just a compilation check
	_, err := NewBrowser(http.NotFoundHandler(), "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
}
