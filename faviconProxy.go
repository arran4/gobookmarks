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

type FavIcon struct {
	Data        []byte
	ContentType string
}

var (
	FaviconCache = struct {
		sync.RWMutex
		cache map[string]*FavIcon
	}{cache: make(map[string]*FavIcon)}
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
		w.Header().Set("Content-Type", cacheValue.ContentType)
		_, _ = w.Write(cacheValue.Data)
		return
	}

	// Fetch the root page content
	rootPageContent, err := fetchURL(urlParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching root page: %s", err), http.StatusInternalServerError)
		return
	}

	// Find the favicon URL from the root page content
	faviconURL, fileType, err := findFaviconURL(rootPageContent, up)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error finding favicon URL: %s", err), http.StatusInternalServerError)
		return
	}

	if fileType == "" {
		fileType = "image/x-icon"
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
	cacheFavicon(urlParam, faviconContent, fileType)

	// Serve the favicon content
	w.Header().Set("Content-Type", fileType)
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

func findFaviconURL(pageContent []byte, baseURL *url.URL) (string, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(pageContent)))
	if err != nil {
		return "", "", err
	}

	var faviconPath string
	var fileType string

	// Find the favicon URL from the meta tags
	doc.Find("link[rel='icon'], link[rel='shortcut icon'], link[rel='alternate icon'], link[id='favicon']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			faviconPath = href
			fileType, _ = s.Attr("type")
			return
		}
	})

	if faviconPath == "" {
		faviconPath = "/favicon.ico"
	}

	p, err := baseURL.Parse(faviconPath)
	if err != nil {
		return "", "", err
	}

	return p.String(), fileType, nil
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

func cacheFavicon(urlParam string, content []byte, contentType string) {
	FaviconCache.Lock()
	defer FaviconCache.Unlock()

	// Delete random keys until the cache is small enough
	for len(FaviconCache.cache) > 1000 { // TODO expose as a config option
		for key := range FaviconCache.cache {
			delete(FaviconCache.cache, key)
			break
		}
	}

	FaviconCache.cache[urlParam] = &FavIcon{
		Data:        content,
		ContentType: contentType,
	}
}

func getCacheFavicon(urlParam string) *FavIcon {
	FaviconCache.Lock()
	defer FaviconCache.Unlock()

	return FaviconCache.cache[urlParam]
}
