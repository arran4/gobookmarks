package gobookmarks

import "net/http"

func CSPMiddleware(csp string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if csp != "" {
				w.Header().Set("Content-Security-Policy", csp)
			}
			next.ServeHTTP(w, r)
		})
	}
}
