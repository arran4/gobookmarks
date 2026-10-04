package gobookmarks

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONRoundTrip_NamesPreserved(t *testing.T) {
	input := "Tab: A\nPage: B\nColumn: Alpha\nCategory: C\nColumn: Beta\nCategory: D\nColumn\nCategory: E"

	list := ParseBookmarks(input)

	jdata := list.ToJSON()

	bytes, err := json.Marshal(jdata)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !strings.Contains(string(bytes), `"name":"Alpha"`) || !strings.Contains(string(bytes), `"name":"Beta"`) {
		t.Fatalf("JSON output does not contain expected column names: %s", string(bytes))
	}

	var reparsed []*JSONTab
	if err := json.Unmarshal(bytes, &reparsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	newList, err := BookmarkListFromJSON(reparsed)
	if err != nil {
		t.Fatalf("BookmarkListFromJSON failed: %v", err)
	}

	out := newList.String()
	expected := "Tab: A\nPage: B\nColumn: Alpha\nCategory: C\nColumn: Beta\nCategory: D\nColumn\nCategory: E\n"

	if out != expected {
		t.Fatalf("Expected:\n%s\nGot:\n%s", expected, out)
	}
}

func TestJSONRoundTrip_LossyWhitespaceRejected(t *testing.T) {
	inputJSON := `[{"name":"A","pages":[{"name":"B","blocks":[{"hr":false,"columns":[{"name":"  \t ","categories":[{"name":"C"}]}]}]}]}]`

	var reparsed []*JSONTab
	if err := json.Unmarshal([]byte(inputJSON), &reparsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	_, err := BookmarkListFromJSON(reparsed)
	if err == nil || !strings.Contains(err.Error(), "lossy shape") {
		t.Fatalf("Expected lossy whitespace name to fail validation, got %v", err)
	}
}

func TestJSONRoundTrip_EmptyNamedColumnPreserved(t *testing.T) {
	input := "Tab: A\nPage: B\nColumn: Empty\nColumn: Second\nCategory: Cat\n"

	list := ParseBookmarks(input)
	jdata := list.ToJSON()
	newList, err := BookmarkListFromJSON(jdata)
	if err != nil {
		t.Fatalf("BookmarkListFromJSON failed: %v", err)
	}

	out := newList.String()
	if out != input {
		t.Fatalf("Expected:\n%s\nGot:\n%s", input, out)
	}
}
