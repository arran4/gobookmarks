package gobookmarks

import (
	"net/http"
	"strconv"
	"testing"
)

func TestPageEditAddMode(t *testing.T) {
	req, _ := http.NewRequest("GET", "/editPage?edit=1&ref=refs/heads/main&tab=0", nil)
	pageIdx, pageErr := strconv.Atoi(req.URL.Query().Get("page"))

	if pageErr == nil {
		t.Errorf("Expected an error since 'page' is not in the URL")
	}

	if pageIdx != 0 {
		t.Errorf("Expected pageIdx to default to 0")
	}

	req2, _ := http.NewRequest("GET", "/editPage?edit=1&ref=refs/heads/main&tab=0&page=0", nil)
	pageIdx2, pageErr2 := strconv.Atoi(req2.URL.Query().Get("page"))

	if pageErr2 != nil {
		t.Errorf("Expected no error since 'page' is 0")
	}
	if pageIdx2 != 0 {
		t.Errorf("Expected pageIdx to be 0")
	}
}
