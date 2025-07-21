package middleware

import "net/http"

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Force HTTPS in production (only set if not in development)
		if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Prevent XSS attacks
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Prevent information leakage
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Prevent Adobe Flash and PDF plugins from loading
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		next.ServeHTTP(w, r)
	})
}
