package gobookmarks

import (
	"net/http/httptest"
	"testing"
)

func TestLoginPageURL(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "root", path: "/", want: "/login"},
		{name: "nested page", path: "/tab/2?page=3", want: "/login?redirect=%2Ftab%2F2%3Fpage%3D3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			loginPageURL := NewFuncs(req)["LoginPageURL"].(func() string)
			if got := loginPageURL(); got != tt.want {
				t.Fatalf("LoginPageURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
