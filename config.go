package gobookmarks

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var (
	Config Configuration
)

const (
	DefaultFaviconCacheSize     int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconMaxCacheCount int           = 1000
	DefaultFaviconCacheMaxAge   time.Duration = 24 * time.Hour
	DefaultCommitsPerPage       int           = 100
)

type Environment interface {
	Getenv(key string) string
	Setenv(key, value string) error
	Geteuid() int
}

type DefaultEnvironment struct{}

func (DefaultEnvironment) Getenv(key string) string       { return os.Getenv(key) }
func (DefaultEnvironment) Setenv(key, value string) error { return os.Setenv(key, value) }
func (DefaultEnvironment) Geteuid() int                   { return os.Geteuid() }

type FileReader interface {
	ReadFile(name string) ([]byte, error)
	Open(name string) (io.ReadCloser, error)
	Stat(name string) (os.FileInfo, error)
}

type DefaultFileReader struct{}

func (DefaultFileReader) ReadFile(name string) ([]byte, error)    { return os.ReadFile(name) }
func (DefaultFileReader) Open(name string) (io.ReadCloser, error) { return os.Open(name) }
func (DefaultFileReader) Stat(name string) (os.FileInfo, error)   { return os.Stat(name) }

// Configuration holds runtime configuration values.
type Configuration struct {
	GithubClientID       string   `json:"github_client_id"`
	GithubSecret         string   `json:"github_secret"`
	GitlabClientID       string   `json:"gitlab_client_id"`
	GitlabSecret         string   `json:"gitlab_secret"`
	ExternalURL          string   `json:"external_url"`
	CSSColumns           bool     `json:"css_columns"`
	DevMode              *bool    `json:"dev_mode"`
	Namespace            string   `json:"namespace"`
	Title                string   `json:"title"`
	GithubServer         string   `json:"github_server"`
	GitlabServer         string   `json:"gitlab_server"`
	FaviconCacheDir      string   `json:"favicon_cache_dir"`
	FaviconCacheSize     int64    `json:"favicon_cache_size"`
	FaviconMaxCacheCount int      `json:"favicon_max_cache_count"`
	LocalGitPath         string   `json:"local_git_path"`
	NoFooter             bool     `json:"no_footer"`
	SessionKey           string   `json:"session_key"`
	SessionName          string   `json:"session_name"`
	DBConnectionProvider string   `json:"db_connection_provider"`
	DBConnectionString   string   `json:"db_connection_string"`
	ProviderOrder        []string `json:"provider_order"`
	CommitsPerPage       int      `json:"commits_per_page"`
}

func (c Configuration) GetDevMode() bool {
	if c.DevMode != nil {
		return *c.DevMode
	}
	return strings.EqualFold(version, "dev")
}

func (c Configuration) GetRepoName() string {
	ns := c.Namespace
	if c.GetDevMode() {
		if ns == "" {
			ns = version
		}
	}

	name := "MyBookmarks"
	if ns != "" {
		name += "-" + ns
	}
	return name
}

func (c Configuration) GetOauthRedirectURL() string {
	externalURL := strings.TrimRight(c.ExternalURL, "/")
	return JoinURL(externalURL, "oauth2Callback")
}

func (c Configuration) GetSessionName() string {
	if c.SessionName != "" {
		return c.SessionName
	}
	return "gobookmarks"
}

// LoadConfigFile loads configuration from the given path.
// It returns the loaded Configuration, a boolean indicating if the file existed,
// and any error that occurred while reading or parsing the file.
func LoadConfigFile(path string, ops ...any) (Configuration, bool, error) {
	var reader FileReader = DefaultFileReader{}
	for _, opt := range ops {
		if r, ok := opt.(FileReader); ok {
			reader = r
		}
	}
	var c Configuration

	log.Printf("attempting to load config from %s", path)

	data, err := reader.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("config file %s not found", path)
			return c, false, nil
		}
		return c, false, fmt.Errorf("unable to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return c, true, fmt.Errorf("unable to parse config file: %w", err)
	}

	log.Printf("successfully loaded config from %s (keys: %s)", path, strings.Join(loadedConfigKeys(c), ", "))

	return c, true, nil
}

func loadedConfigKeys(c Configuration) []string {
	var keys []string
	v := reflect.ValueOf(c)
	t := reflect.TypeOf(c)
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsZero() {
			key := t.Field(i).Tag.Get("json")
			if key == "" {
				key = t.Field(i).Name
			}
			keys = append(keys, key)
		}
	}
	return keys
}

// LoadConfigFileInto loads configuration from the given path directly onto a struct.
// It modifies `c` and returns a boolean indicating if the file existed,
// and any error that occurred while reading or parsing the file.
func LoadConfigFileInto(c *Configuration, path string, ops ...any) (bool, error) {
	var reader FileReader = DefaultFileReader{}
	for _, opt := range ops {
		if r, ok := opt.(FileReader); ok {
			reader = r
		}
	}
	log.Printf("attempting to load config from %s", path)

	data, err := reader.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("config file %s not found", path)
			return false, nil
		}
		return false, fmt.Errorf("unable to read config file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return true, fmt.Errorf("unable to parse config file: %w", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		var keys []string
		for k := range parsed {
			keys = append(keys, k)
		}
		log.Printf("successfully loaded config from %s (keys: %s)", path, strings.Join(keys, ", "))
	} else {
		log.Printf("successfully loaded config from %s", path)
	}

	return true, nil
}

// MergeConfig copies values from src into dst if they are non-zero.
func MergeConfig(dst *Configuration, src Configuration) {
	if src.GithubClientID != "" {
		dst.GithubClientID = src.GithubClientID
	}
	if src.GithubSecret != "" {
		dst.GithubSecret = src.GithubSecret
	}
	if src.GitlabClientID != "" {
		dst.GitlabClientID = src.GitlabClientID
	}
	if src.GitlabSecret != "" {
		dst.GitlabSecret = src.GitlabSecret
	}
	if src.ExternalURL != "" {
		dst.ExternalURL = src.ExternalURL
	}
	if src.CSSColumns {
		dst.CSSColumns = true
	}
	if src.DevMode != nil {
		dst.DevMode = src.DevMode
	}
	if src.Namespace != "" {
		dst.Namespace = src.Namespace
	}
	if src.Title != "" {
		dst.Title = src.Title
	}
	if src.GithubServer != "" {
		dst.GithubServer = src.GithubServer
	}
	if src.GitlabServer != "" {
		dst.GitlabServer = src.GitlabServer
	}
	if src.FaviconCacheDir != "" {
		dst.FaviconCacheDir = src.FaviconCacheDir
	}
	if src.FaviconCacheSize != 0 {
		dst.FaviconCacheSize = src.FaviconCacheSize
	}
	if src.FaviconMaxCacheCount != 0 {
		dst.FaviconMaxCacheCount = src.FaviconMaxCacheCount
	}
	if src.LocalGitPath != "" {
		dst.LocalGitPath = src.LocalGitPath
	}
	if src.NoFooter {
		dst.NoFooter = true
	}
	if src.SessionKey != "" {
		dst.SessionKey = src.SessionKey
	}
	if src.SessionName != "" {
		dst.SessionName = src.SessionName
	}
	if src.DBConnectionProvider != "" {
		dst.DBConnectionProvider = src.DBConnectionProvider
	}
	if src.DBConnectionString != "" {
		dst.DBConnectionString = src.DBConnectionString
	}
	if src.CommitsPerPage != 0 {
		dst.CommitsPerPage = src.CommitsPerPage
	}
	if len(src.ProviderOrder) > 0 {
		dst.ProviderOrder = append([]string(nil), src.ProviderOrder...)
	}
}

// DefaultConfigPath returns the path to the config file depending on
// environment and the effective user. If running as a non-root user and
// XDG variables are set, the config lives under the XDG config directory.
// Otherwise it falls back to /etc/gobookmarks/config.json.
func DefaultConfigPath(ops ...any) string {
	var env Environment = DefaultEnvironment{}
	for _, opt := range ops {
		_ = opt // To silence unused warning if empty block
		if e, ok := opt.(Environment); ok {
			env = e
		}
	}
	if p := env.Getenv("GOBM_CONFIG_FILE"); p != "" {
		return p
	}
	if env.Geteuid() != 0 {
		xdg := env.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "gobookmarks", "config.json")
		}
		if home := env.Getenv("HOME"); home != "" {
			return filepath.Join(home, ".config", "gobookmarks", "config.json")
		}
	}
	return "/etc/gobookmarks/config.json"
}

// DefaultSessionKeyPath returns the location of the session key file.
// User installs store it under XDG state or ~/.local/state. System-wide
// installations use /var/lib/gobookmarks/session.key.
// DefaultSessionKeyPath returns the path used to read or write the
// session key depending on the value of writing. When writing it
// chooses the path appropriate for the current user. When reading it
// checks the usual locations and returns the first existing file,
// falling back to the writing location if none are found.
func DefaultSessionKeyPath(writing bool, ops ...any) string {
	var env Environment = DefaultEnvironment{}
	for _, opt := range ops {
		_ = opt // To silence unused warning if empty block
		if e, ok := opt.(Environment); ok {
			env = e
		}
	}
	var userPaths []string
	if xdg := env.Getenv("XDG_STATE_HOME"); xdg != "" {
		userPaths = append(userPaths, filepath.Join(xdg, "gobookmarks", "session.key"))
	}
	if home := env.Getenv("HOME"); home != "" {
		userPaths = append(userPaths, filepath.Join(home, ".local", "state", "gobookmarks", "session.key"))
	}

	systemPath := "/var/lib/gobookmarks/session.key"

	if !writing {
		if env.Geteuid() == 0 {
			if fileExists(systemPath, ops...) {
				return systemPath
			}
			for _, p := range userPaths {
				if fileExists(p, ops...) {
					return p
				}
			}
		} else {
			for _, p := range userPaths {
				if fileExists(p, ops...) {
					return p
				}
			}
			if fileExists(systemPath, ops...) {
				return systemPath
			}
		}
	}

	if env.Geteuid() != 0 {
		if len(userPaths) > 0 {
			return userPaths[0]
		}
	}
	return systemPath
}

func fileExists(path string, ops ...any) bool {
	var reader FileReader = DefaultFileReader{}
	for _, opt := range ops {
		if r, ok := opt.(FileReader); ok {
			reader = r
		}
	}
	_, err := reader.Stat(path)
	return err == nil
}

// Lines should be in KEY=VALUE format and may be commented with '#'.
func LoadEnvFile(path string, ops ...any) error {
	var env Environment = DefaultEnvironment{}
	var reader FileReader = DefaultFileReader{}
	for _, opt := range ops {
		if e, ok := opt.(Environment); ok {
			env = e
		}
		if r, ok := opt.(FileReader); ok {
			reader = r
		}
	}
	data, err := reader.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if env.Getenv(key) == "" {
			_ = env.Setenv(key, val)
		}
	}
	return scanner.Err()
}
