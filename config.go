package gobookmarks

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config holds runtime configuration values.
type Config struct {
	GithubClientID   string `json:"github_client_id"`
	GithubSecret     string `json:"github_secret"`
	GitlabClientID   string `json:"gitlab_client_id"`
	GitlabSecret     string `json:"gitlab_secret"`
	ExternalURL      string `json:"external_url"`
	CssColumns       bool   `json:"css_columns"`
	Namespace        string `json:"namespace"`
	Title            string `json:"title"`
	GithubServer     string `json:"github_server"`
	GitlabServer     string `json:"gitlab_server"`
	FaviconCacheDir  string `json:"favicon_cache_dir"`
	FaviconCacheSize int64  `json:"favicon_cache_size"`
	JSONDBPath       string `json:"json_db_path"`
}

// LoadConfigFile loads configuration from the given path if it exists.
func LoadConfigFile(path string) (Config, error) {
	var c Config
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	err = json.Unmarshal(data, &c)
	return c, err
}

// MergeConfig copies values from src into dst if they are non-zero.
func MergeConfig(dst *Config, src Config) {
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
	if src.CssColumns {
		dst.CssColumns = true
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
	if src.JSONDBPath != "" {
		dst.JSONDBPath = src.JSONDBPath
	}
}

// DefaultConfigPath returns the path to the config file depending on
// environment and the effective user. If running as a non-root user and
// XDG variables are set, the config lives under the XDG config directory.
// Otherwise it falls back to /etc/gobookmarks/config.json.
func DefaultConfigPath() string {
	if p := os.Getenv("GOBM_CONFIG_FILE"); p != "" {
		return p
	}
	if os.Geteuid() != 0 {
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "gobookmarks", "config.json")
		}
		if home := os.Getenv("HOME"); home != "" {
			return filepath.Join(home, ".config", "gobookmarks", "config.json")
		}
	}
	return "/etc/gobookmarks/config.json"
}

// Lines should be in KEY=VALUE format and may be commented with '#'.
func LoadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
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
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}
