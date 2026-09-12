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

func TestIsSafeRedirect(t *testing.T) {
	tests := []struct {
		input string
		safe  bool
	}{
		{"/tab/2?page=3", true},
		{"/", true},
		{"/tab/2", true},
		{"/bookmarks/mine?sort=name&order=asc", true},
		{"https://evil.com", false},
		{"http://evil.com", false},
		{"//evil.com", false},
		{"//evil.com/path", false},
		{"/\\evil.com", false},
		{"\\evil.com", false},
		{"javascript:alert(1)", false},
		{"data:text/html,evil", false},
		{"/evil\r\nSet-Cookie:bad=true", false},
		{"/evil\nSet-Cookie:bad=true", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsSafeRedirect(tt.input); got != tt.safe {
			t.Errorf("IsSafeRedirect(%q) = %v; want %v", tt.input, got, tt.safe)
		}
	}
}
