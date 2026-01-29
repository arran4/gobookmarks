package main

import (
	"testing"
)

func TestLoadConfigUsesExternalURL(t *testing.T) {
	rc := NewRootCommand()

	t.Setenv("EXTERNAL_URL", "http://example.com/app")

	if err := rc.loadConfig(); err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if rc.cfg.ExternalURL != "http://example.com/app" {
		t.Fatalf("external url not loaded from env: %q", rc.cfg.ExternalURL)
	}
}
