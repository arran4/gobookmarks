package gobookmarks

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

// Config holds runtime configuration values.
type Config struct {
	Oauth2ClientID string `json:"oauth2_client_id"`
	Oauth2Secret   string `json:"oauth2_secret"`
	ExternalURL    string `json:"external_url"`
	CssColumns     bool   `json:"css_columns"`
	Namespace      string `json:"namespace"`
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
	if src.Oauth2ClientID != "" {
		dst.Oauth2ClientID = src.Oauth2ClientID
	}
	if src.Oauth2Secret != "" {
		dst.Oauth2Secret = src.Oauth2Secret
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
}

// LoadEnvFile sets environment variables from the given file if they are not already defined.
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
