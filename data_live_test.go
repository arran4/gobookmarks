//go:build live

package gobookmarks

import (
	"bytes"
	"os"
	"testing"
)

func TestAppJSLive(t *testing.T) {
	liveData := GetAppJSData()

	// Try to read it directly from disk
	fsPath := "web/app.mjs"
	if _, err := os.Stat(fsPath); os.IsNotExist(err) {
		fsPath = "../../web/app.mjs"
	}
	if _, err := os.Stat(fsPath); os.IsNotExist(err) {
		fsPath = "../web/app.mjs"
	}

	diskData, err := os.ReadFile(fsPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", fsPath, err)
	}

	if !bytes.Equal(liveData, diskData) {
		t.Errorf("live mode GetAppJSData() did not return exact file bytes")
	}
}
