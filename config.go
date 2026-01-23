package gobookmarks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// LoadConfigFile loads configuration from the given path.
// It returns the loaded Configuration, a boolean indicating if the file existed,
// and any error that occurred while reading or parsing the file.
func LoadConfigFile(path string) (*Configuration, bool, error) {
	c := NewConfiguration()

	log.Printf("attempting to load config from %s", path)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("config file %s not found", path)
			return c, false, nil
		}
		return c, false, fmt.Errorf("unable to read config file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return c, true, fmt.Errorf("unable to parse config file: %w", err)
	}

	log.Printf("successfully loaded config from %s (keys: %s)", path, strings.Join(loadedConfigKeys(c), ", "))

	return c, true, nil
}

func loadedConfigKeys(c *Configuration) []string {
	var keys []string
	v := reflect.ValueOf(c).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		// Skip "data" field or others that don't have json tag or are not exported
		if !v.Field(i).CanInterface() {
			continue
		}
		if !v.Field(i).IsZero() {
			key := t.Field(i).Tag.Get("json")
			if key == "-" {
				continue
			}
			if key == "" {
				key = t.Field(i).Name
			}
			keys = append(keys, key)
		}
	}
	return keys
}

// MergeConfig copies values from src into dst if they are non-zero.
func MergeConfig(dst *Configuration, src *Configuration) {
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
	if src.LocalGitPath != "" {
		dst.LocalGitPath = src.LocalGitPath
	}
	if src.NoFooter {
		dst.NoFooter = true
	}
	if src.SessionKey != "" {
		dst.SessionKey = src.SessionKey
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

	// Copy data map
	if src.data != nil {
		if dst.data == nil {
			dst.data = make(map[string]string)
		}
		for k, v := range src.data {
			dst.data[k] = v
		}
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

// DefaultSessionKeyPath returns the location of the session key file.
// User installs store it under XDG state or ~/.local/state. System-wide
// installations use /var/lib/gobookmarks/session.key.
// DefaultSessionKeyPath returns the path used to read or write the
// session key depending on the value of writing. When writing it
// chooses the path appropriate for the current user. When reading it
// checks the usual locations and returns the first existing file,
// falling back to the writing location if none are found.
func DefaultSessionKeyPath(writing bool) string {
	var userPaths []string
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		userPaths = append(userPaths, filepath.Join(xdg, "gobookmarks", "session.key"))
	}
	if home := os.Getenv("HOME"); home != "" {
		userPaths = append(userPaths, filepath.Join(home, ".local", "state", "gobookmarks", "session.key"))
	}

	systemPath := "/var/lib/gobookmarks/session.key"

	if !writing {
		if os.Geteuid() == 0 {
			if fileExists(systemPath) {
				return systemPath
			}
			for _, p := range userPaths {
				if fileExists(p) {
					return p
				}
			}
		} else {
			for _, p := range userPaths {
				if fileExists(p) {
					return p
				}
			}
			if fileExists(systemPath) {
				return systemPath
			}
		}
	}

	if os.Geteuid() != 0 {
		if len(userPaths) > 0 {
			return userPaths[0]
		}
	}
	return systemPath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
