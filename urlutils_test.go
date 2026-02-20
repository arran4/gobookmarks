package gobookmarks

import "testing"

func TestJoinURL(t *testing.T) {
	tests := []struct {
		base string
		elem string
		want string
	}{
		{"http://example.com", "oauth2Callback", "http://example.com/oauth2Callback"},
		{"http://example.com/", "oauth2Callback", "http://example.com/oauth2Callback"},
		{"http://example.com///", "oauth2Callback", "http://example.com/oauth2Callback"},
		{"http://example.com/base", "oauth2Callback", "http://example.com/base/oauth2Callback"},
		{"http://example.com/base/", "/oauth2Callback///", "http://example.com/base/oauth2Callback"},
		{"", "///oauth2Callback///", "/oauth2Callback"},
	}
	for _, tt := range tests {
		got := JoinURL(tt.base, tt.elem)
		if got != tt.want {
			t.Fatalf("JoinURL(%q, %q) = %q, want %q", tt.base, tt.elem, got, tt.want)
		}
	}
}
