package gobookmarks

import (
	"net/http"
	"testing"
)

func TestTabFromRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "/editTab?tab=1", nil)
	idx, has := TabFromRequest(req)
	if !has || idx != 1 {
		t.Errorf("Expected tab index 1 and has true, got %d, %v", idx, has)
	}

	req2, _ := http.NewRequest("GET", "/editTab", nil)
	idx2, has2 := TabFromRequest(req2)
	if has2 || idx2 != 0 {
		t.Errorf("Expected tab index 0 and has false, got %d, %v", idx2, has2)
	}
}

func TestEditTabAddMode(t *testing.T) {
	req, _ := http.NewRequest("GET", "/editTab?edit=1", nil)
	tabName := req.URL.Query().Get("name")
	_, hasTab := TabFromRequest(req)
	isAddMode := !hasTab && tabName == ""
	if !isAddMode {
		t.Errorf("Expected isAddMode to be true")
	}

	req2, _ := http.NewRequest("GET", "/editTab?edit=1&tab=0", nil)
	tabName2 := req2.URL.Query().Get("name")
	_, hasTab2 := TabFromRequest(req2)
	isAddMode2 := !hasTab2 && tabName2 == ""
	if isAddMode2 {
		t.Errorf("Expected isAddMode to be false when tab is present")
	}
}
