package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with security middleware
	wrappedHandler := SecurityHeaders(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute the handler
	wrappedHandler.ServeHTTP(w, req)

	// Check that the response is successful
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that security headers are set
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expectedValue := range expectedHeaders {
		if actualValue := w.Header().Get(header); actualValue != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

func TestSecurityHeadersWithConfig(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create custom config
	config := SecurityConfig{
		EnableHSTS:      true,
		HSTSMaxAge:      3600,
		EnableHTTPSOnly: true,
	}

	// Wrap with security middleware with config
	wrappedHandler := SecurityHeadersWithConfig(config)(testHandler)

	// Test with HTTPS request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Check that security headers are set
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options header should be set")
	}

	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS header should be set for HTTPS requests")
	}
}

func TestSecurityHeadersHTTPRequest(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with security middleware
	wrappedHandler := SecurityHeaders(testHandler)

	// Test HTTP request (no HTTPS headers)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Basic security headers should still be set
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options header should be set")
	}

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options header should be set")
	}
}

func TestSecurityHeadersPreservesResponse(t *testing.T) {
	// Create a test handler that returns specific content
	expectedBody := "test response content"
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	})

	// Wrap with security middleware
	wrappedHandler := SecurityHeaders(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute the handler
	wrappedHandler.ServeHTTP(w, req)

	// Check that the original response is preserved
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, w.Body.String())
	}

	// Check that both security headers and original headers are present
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Error("Original Content-Type header should be preserved")
	}

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Security headers should be added")
	}
}
