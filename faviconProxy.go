package gobookmarks

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/image/draw"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// fetchUserAgent is used when fetching pages and icons so sites return
// favicon links consistently (e.g. Google Calendar).
const fetchUserAgent = "Mozilla/5.0 (compatible; gobookmarks/1.0)"

type FavIcon struct {
	Data        []byte
	ContentType string
}

type diskMeta struct {
	ContentType string    `json:"content_type"`
	Expiry      time.Time `json:"expiry"`
}

var defaultFaviconProxy = NewFaviconProxy(&Config)

func NewFaviconProxy(cfg *Configuration) *FaviconProxy {
	return &FaviconProxy{
		Config: cfg,
		cache:  make(map[string]*FavIcon),
		now:    time.Now,
	}
}

type FaviconProxy struct {
	Config *Configuration
	mu     sync.RWMutex
	cache  map[string]*FavIcon
	now    func() time.Time
}

func (fp *FaviconProxy) cacheFileBase(u string) string {
	h := sha1.Sum([]byte(strings.ToLower(u)))
	return filepath.Join(fp.Config.FaviconCacheDir, hex.EncodeToString(h[:]))
}

func (fp *FaviconProxy) readDiskFavicon(u string) *FavIcon {
	if fp.Config.FaviconCacheDir == "" {
		return nil
	}
	base := fp.cacheFileBase(u)
	dataPath := base + ".dat"
	metaPath := base + ".json"

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil
	}
	var m diskMeta
	if json.Unmarshal(metaBytes, &m) != nil {
		return nil
	}
	if fp.now().After(m.Expiry) {
		_ = os.Remove(dataPath)
		_ = os.Remove(metaPath)
		return nil
	}
	b, err := os.ReadFile(dataPath)
	if err != nil {
		return nil
	}
	return &FavIcon{Data: b, ContentType: m.ContentType}
}

func (fp *FaviconProxy) writeDiskFavicon(u string, f *FavIcon, expiry time.Time) {
	if fp.Config.FaviconCacheDir == "" {
		return
	}
	if err := os.MkdirAll(fp.Config.FaviconCacheDir, 0o755); err != nil {
		return
	}
	base := fp.cacheFileBase(u)
	dataPath := base + ".dat"
	metaPath := base + ".json"
	_ = os.WriteFile(dataPath, f.Data, 0o644)
	m := diskMeta{ContentType: f.ContentType, Expiry: expiry}
	mb, _ := json.Marshal(m)
	_ = os.WriteFile(metaPath, mb, 0o644)
	fp.enforceCacheLimit()
}

func (fp *FaviconProxy) enforceCacheLimit() {
	if fp.Config.FaviconCacheDir == "" || fp.Config.FaviconCacheSize <= 0 {
		return
	}
	entries, err := filepath.Glob(filepath.Join(fp.Config.FaviconCacheDir, "*.dat"))
	if err != nil {
		return
	}
	type info struct {
		path string
		mod  time.Time
		size int64
	}
	var list []info
	var total int64
	for _, p := range entries {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		list = append(list, info{path: p, mod: fi.ModTime(), size: fi.Size()})
		total += fi.Size()
	}
	sort.Slice(list, func(i, j int) bool { return list[i].mod.Before(list[j].mod) })
	for total > fp.Config.FaviconCacheSize && len(list) > 0 {
		fi := list[0]
		list = list[1:]
		total -= fi.size
		_ = os.Remove(fi.path)
		_ = os.Remove(strings.TrimSuffix(fi.path, ".dat") + ".json")
	}
}

func FaviconProxyHandler(w http.ResponseWriter, r *http.Request) {
	defaultFaviconProxy.ServeHTTP(w, r)
}

func (fp *FaviconProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	sizeParam := r.URL.Query().Get("size")
	size := 0
	if sizeParam != "" {
		if i, err := strconv.Atoi(sizeParam); err == nil && i > 0 {
			size = i
		}
	}

	targetKey := urlParam
	if size > 0 {
		targetKey = fmt.Sprintf("%s#size=%d", urlParam, size)
	}

	getFromCache := func(key string) *FavIcon {
		val := fp.getCacheFavicon(key)
		if fp.Config.FaviconCacheDir != "" && val != nil {
			if fp.readDiskFavicon(key) == nil {
				fp.removeCacheFavicon(key)
				val = nil
			}
		}
		if val == nil && fp.Config.FaviconCacheDir != "" {
			if diskVal := fp.readDiskFavicon(key); diskVal != nil {
				val = diskVal
				fp.cacheFavicon(key, diskVal.Data, diskVal.ContentType)
			}
		}
		return val
	}

	// 1. Try exact match (e.g. resized)
	if size > 0 {
		if icon := getFromCache(targetKey); icon != nil {
			w.Header().Set("Content-Type", icon.ContentType)
			_, _ = w.Write(icon.Data)
			return
		}
	}

	// 2. Try base match
	cacheValue := getFromCache(urlParam)
	if cacheValue != nil {
		icon := cacheValue
		if size > 0 {
			if data, ct, err := resizeImage(cacheValue.Data, size); err == nil {
				icon = &FavIcon{Data: data, ContentType: ct}
				fp.cacheFavicon(targetKey, data, ct)
				if fp.Config.FaviconCacheDir != "" {
					// We use DefaultFaviconCacheMaxAge for resized items derived from cache
					fp.writeDiskFavicon(targetKey, icon, fp.now().Add(DefaultFaviconCacheMaxAge))
				}
			}
		}
		w.Header().Set("Content-Type", icon.ContentType)
		_, _ = w.Write(icon.Data)
		return
	}

	// Fetch the root page content
	rootPageContent, err := fetchURL(r.Context(), urlParam)
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
	faviconContent, hdr, err := downloadURL(r.Context(), faviconURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error proxying favicon: %s", err), http.StatusInternalServerError)
		return
	}

	if len(faviconContent) > 1*1024*1024 {
		http.Error(w, fmt.Sprintf("Error proxying favicon: %s", "favicon too large"), http.StatusInternalServerError)
		return
	}

	// Cache the favicon content
	expiry := fp.now().Add(DefaultFaviconCacheMaxAge)
	if cc := hdr.Get("Cache-Control"); cc != "" {
		if strings.Contains(cc, "max-age=") {
			parts := strings.Split(cc, "=")
			if len(parts) == 2 {
				if sec, err := strconv.Atoi(parts[1]); err == nil {
					expiry = fp.now().Add(time.Duration(sec) * time.Second)
				}
			}
		}
	}
	fp.cacheFavicon(urlParam, faviconContent, fileType)
	if fp.Config.FaviconCacheDir != "" {
		fp.writeDiskFavicon(urlParam, &FavIcon{Data: faviconContent, ContentType: fileType}, expiry)
	}

	// Serve the favicon content
	icon := &FavIcon{Data: faviconContent, ContentType: fileType}
	if size > 0 {
		if data, ct, err := resizeImage(icon.Data, size); err == nil {
			icon = &FavIcon{Data: data, ContentType: ct}
			fp.cacheFavicon(targetKey, data, ct)
			if fp.Config.FaviconCacheDir != "" {
				fp.writeDiskFavicon(targetKey, icon, expiry)
			}
		}
	}
	w.Header().Set("Content-Type", icon.ContentType)
	_, _ = w.Write(icon.Data)
}

func fetchURL(ctx context.Context, urlParam string) ([]byte, error) {
	req, err := http.NewRequest("GET", urlParam, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fetchUserAgent)
	resp, err := httpClientFromContext(ctx).Do(req)
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
	doc.Find("link[rel='icon'], link[rel='shortcut icon'], link[rel='alternate icon'], link[id='favicon']").Each(func(_ int, s *goquery.Selection) {
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

func downloadURL(ctx context.Context, url string) ([]byte, http.Header, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", fetchUserAgent)
	resp, err := httpClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	b, err := io.ReadAll(resp.Body)
	return b, resp.Header, err
}

// Scenario handlers carry a controlled client in their request context. This
// also makes an unexpected favicon request fail at the same fixture boundary
// as provider requests instead of escaping to the network.
func httpClientFromContext(ctx context.Context) *http.Client {
	if client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && client != nil {
		return client
	}
	return http.DefaultClient
}

func (fp *FaviconProxy) cacheFavicon(urlParam string, content []byte, contentType string) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Delete random keys until the cache is small enough
	for len(fp.cache) > 0 && len(fp.cache) >= fp.Config.FaviconMaxCacheCount {
		for key := range fp.cache {
			delete(fp.cache, key)
			break
		}
	}

	fp.cache[urlParam] = &FavIcon{
		Data:        content,
		ContentType: contentType,
	}
}

func (fp *FaviconProxy) getCacheFavicon(urlParam string) *FavIcon {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	return fp.cache[urlParam]
}

func (fp *FaviconProxy) removeCacheFavicon(urlParam string) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	delete(fp.cache, urlParam)
}

func resizeImage(data []byte, size int) ([]byte, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.ApproxBiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, dst, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	case "gif":
		if err := gif.Encode(&buf, dst, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/gif", nil
	default:
		if err := png.Encode(&buf, dst); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	}
}
