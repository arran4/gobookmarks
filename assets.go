package gobookmarks

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

type Asset struct {
	Bytes []byte
	ETag  string
}

type Registry struct {
	mu           sync.RWMutex
	logicalToURL map[string]string
	urlToAsset   map[string]*Asset
	fsys         fs.FS
	liveMode     bool
}

func NewRegistry(fsys fs.FS, liveMode bool) (*Registry, error) {
	reg := &Registry{
		logicalToURL: make(map[string]string),
		urlToAsset:   make(map[string]*Asset),
		fsys:         fsys,
		liveMode:     liveMode,
	}

	err := reg.Reload()
	return reg, err
}

func (r *Registry) Reload() error {
	if r.fsys == nil {
		return nil
	}
	logicalToURL := make(map[string]string)
	urlToAsset := make(map[string]*Asset)

	publicAssets := []string{"main.css", "logo.png"}

	for _, p := range publicAssets {
		b, err := fs.ReadFile(r.fsys, p)
		if err != nil {
            if !r.liveMode {
                if !strings.Contains(err.Error(), "file does not exist") && !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "not found") {
                    return err
                }
                // We just continue on not found in case some test removes it
                continue
            }
            return err
		}

		// Calculate full SHA-256 digest
		sum := sha256.Sum256(b)
		fullDigest := hex.EncodeToString(sum[:])

		fingerprint := fullDigest

		// Insert fingerprint before extension (e.g., css/main.a1b2c3d4....css)
		ext := path.Ext(p)
		base := strings.TrimSuffix(p, ext)
		fingerprintedURL := fmt.Sprintf("/assets/%s.%s%s", base, fingerprint, ext)

		logicalToURL[p] = fingerprintedURL
		cleanLogical := strings.TrimPrefix(p, "./")
		logicalToURL[cleanLogical] = fingerprintedURL

		urlToAsset[fingerprintedURL] = &Asset{
			Bytes: b,
			ETag:  fmt.Sprintf(`"%s"`, fullDigest),
		}
	}

	r.mu.Lock()
	r.logicalToURL = logicalToURL
	r.urlToAsset = urlToAsset
	r.mu.Unlock()

	return nil
}

func (r *Registry) AssetURL(logicalName string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cleaned := strings.TrimPrefix(logicalName, "/")
	cleaned = strings.TrimPrefix(cleaned, "./")
	if url, ok := r.logicalToURL[cleaned]; ok {
		return url, nil
	}
	if url, ok := r.logicalToURL[logicalName]; ok {
		return url, nil
	}
	return "", fmt.Errorf("asset not found: %s", logicalName)
}

func (r *Registry) GetAsset(url string) (*Asset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.urlToAsset[url]
	return a, ok
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	asset, ok := r.GetAsset(req.URL.Path)
	if !ok {
		http.NotFound(w, req)
		return
	}

	// Set caching headers
	if r.liveMode {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", asset.ETag)
	}

	// Since these are embedded assets, we might not have a real ModTime.
	// We pass time.Time{} and let ETag handle validation.
	http.ServeContent(w, req, req.URL.Path, time.Time{}, bytes.NewReader(asset.Bytes))
}

func GetAssetRegistry() *Registry {
	return GetAssetProvider().(*Registry)
}

type AssetProvider interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request)
	AssetURL(logicalName string) (string, error)
	GetAsset(url string) (*Asset, bool)
}

func AssetURL(logicalName string) (string, error) {
	reg := GetAssetProvider()
	if reg == nil {
		return "", fmt.Errorf("asset provider not initialized")
	}
	return reg.AssetURL(logicalName)
}
