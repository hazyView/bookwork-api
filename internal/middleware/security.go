package middleware

import (
	"net/http"
	"strconv"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	EnableHSTS      bool
	HSTSMaxAge      int
	EnableHTTPSOnly bool
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	config := SecurityConfig{
		EnableHSTS:      true,
		HSTSMaxAge:      31536000,
		EnableHTTPSOnly: false,
	}
	return SecurityHeadersWithConfig(config)(next)
}

// SecurityHeadersWithConfig creates middleware with configurable security headers
func SecurityHeadersWithConfig(config SecurityConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// --- EXISTING HEADERS ---
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Force HTTPS in production
			if config.EnableHSTS && (r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil) {
				w.Header().Set("Strict-Transport-Security", "max-age="+strconv.Itoa(config.HSTSMaxAge)+"; includeSubDomains; preload")
			}

			// Prevent XSS attacks
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Prevent information leakage
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Prevent Adobe Flash and PDF plugins from loading
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

			// --- NEWLY ADDED HEADERS from AUDIT ---
			// A restrictive default CSP for an API. It prevents rendering of the API response as a document.
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none';")

			// Disables features like microphone and camera
			w.Header().Set("Permissions-Policy", "microphone=(), camera=(), geolocation=(), payment=()")

			// Add cross-origin policies for modern security posture
			w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

			// API specific headers
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			next.ServeHTTP(w, r)
		})
	}
}
