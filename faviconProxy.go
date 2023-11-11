package gobookmarks

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	FaviconCache = struct {
		sync.RWMutex
		cache map[string][]byte
	}{cache: make(map[string][]byte)}
)

func FaviconProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the URL parameter
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}

	if urlParam == "" {
		return
	}

	up, err := url.Parse(urlParam)
	if err != nil {
		err := fmt.Errorf("parsing URL: %s", err)
		log.Printf("Error %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	up, _ = up.Parse("/")

	urlParam = strings.ToLower(up.String())

	cacheValue := getCacheFavicon(urlParam)
	if cacheValue != nil {
		w.Header().Set("Content-Type", "image/x-icon")
		_, _ = w.Write(cacheValue)
		return
	}

	// Fetch the root page content
	rootPageContent, err := fetchURL(urlParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching root page: %s", err), http.StatusInternalServerError)
		return
	}

	// Find the favicon URL from the root page content
	faviconURL, err := findFaviconURL(rootPageContent, urlParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error finding favicon URL: %s", err), http.StatusInternalServerError)
		return
	}

	// Proxy the favicon request
	faviconContent, err := downloadUrl(faviconURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error proxying favicon: %s", err), http.StatusInternalServerError)
		return
	}

	if len(faviconContent) > 1*1024*1024 {
		http.Error(w, fmt.Sprintf("Error proxying favicon: %s", "favicon too large"), http.StatusInternalServerError)
		return
	}

	// Cache the favicon content
	cacheFavicon(urlParam, faviconContent)

	// Serve the favicon content
	w.Header().Set("Content-Type", "image/x-icon")
	_, _ = w.Write(faviconContent)
}

func fetchURL(urlParam string) ([]byte, error) {
	resp, err := http.Get(urlParam)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	return io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024+1))
}

func findFaviconURL(pageContent []byte, baseURL string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(pageContent)))
	if err != nil {
		return "", err
	}

	var faviconURL string

	// Find the favicon URL from the meta tags
	doc.Find("link[rel='icon'], link[rel='shortcut icon']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			faviconURL = href
			return
		}
	})

	if faviconURL == "" {
		faviconURL = baseURL + "favicon.ico"
	}

	// Convert relative URLs to absolute URLs
	u, err := url.Parse(faviconURL)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	faviconURL = base.ResolveReference(u).String()

	return faviconURL, nil
}

func downloadUrl(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	return io.ReadAll(resp.Body)
}

func cacheFavicon(urlParam string, content []byte) {
	FaviconCache.Lock()
	defer FaviconCache.Unlock()

	// Delete random keys until the cache is small enough
	for len(FaviconCache.cache) > 1000 { // TODO expose as a config option
		for key := range FaviconCache.cache {
			delete(FaviconCache.cache, key)
			break
		}
	}

	FaviconCache.cache[urlParam] = content
}

func getCacheFavicon(urlParam string) (content []byte) {
	FaviconCache.Lock()
	defer FaviconCache.Unlock()

	return FaviconCache.cache[urlParam]
}
