package gobookmarks

import "net/http"

const ContentSecurityPolicyEnv = "CONTENT_SECURITY_POLICY"

// SecurityHeadersMiddleware sets common security headers such as Content-Security-Policy.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		policy := ContentSecurityPolicy
		if policy == "" {
			policy = DefaultContentSecurityPolicy
		}
		w.Header().Set("Content-Security-Policy", policy)
		next.ServeHTTP(w, r)
	})
}
