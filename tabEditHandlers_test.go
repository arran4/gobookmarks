package gobookmarks

import (
	"net/http"
	"testing"
)

func TestTabFromRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "/editTab?tab=1", nil)
	idx := TabFromRequest(req)
	if idx != 1 {
		t.Errorf("Expected tab index 1, got %d", idx)
	}

	req2, _ := http.NewRequest("GET", "/editTab", nil)
	idx2 := TabFromRequest(req2)
	if idx2 != 0 {
		t.Errorf("Expected tab index 0, got %d", idx2)
	}
}

func TestEditTabAddMode(t *testing.T) {
	req, _ := http.NewRequest("GET", "/editTab?edit=1", nil)
	tabName := req.URL.Query().Get("name")
	isAddMode := !req.URL.Query().Has("tab") && tabName == ""
	if !isAddMode {
		t.Errorf("Expected isAddMode to be true")
	}

	req2, _ := http.NewRequest("GET", "/editTab?edit=1&tab=0", nil)
	tabName2 := req2.URL.Query().Get("name")
	isAddMode2 := !req2.URL.Query().Has("tab") && tabName2 == ""
	if isAddMode2 {
		t.Errorf("Expected isAddMode to be false when tab is present")
	}
}
