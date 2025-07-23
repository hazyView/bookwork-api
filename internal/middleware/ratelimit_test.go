package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	// Test creating a new rate limiter
	limiter := NewRateLimiter(10, time.Minute)

	if limiter == nil {
		t.Fatal("Rate limiter should not be nil")
	}

	if limiter.limit != 10 {
		t.Errorf("Expected limit 10, got %d", limiter.limit)
	}

	if limiter.window != time.Minute {
		t.Errorf("Expected window 1 minute, got %v", limiter.window)
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create rate limiter with 2 requests per second
	limiter := NewRateLimiter(2, time.Second)

	// Wrap with rate limit middleware
	wrappedHandler := limiter.Middleware(testHandler)

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request should succeed, got status %d", w.Code)
	}

	// Second request from same IP should succeed
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	w2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request should succeed, got status %d", w2.Code)
	}

	// Third request from same IP should be rate limited
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	w3 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Third request should be rate limited, got status %d", w3.Code)
	}
}

func TestRateLimiterWithXRealIP(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create rate limiter with 1 request per second
	limiter := NewRateLimiter(1, time.Second)
	wrappedHandler := limiter.Middleware(testHandler)

	// Test with X-Real-IP header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Request with X-Real-IP should succeed, got status %d", w.Code)
	}

	// Second request from same IP should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Real-IP", "192.168.1.100")
	w2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status %d", w2.Code)
	}
}

func TestRateLimiterWithXForwardedFor(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create rate limiter with 1 request per second
	limiter := NewRateLimiter(1, time.Second)
	wrappedHandler := limiter.Middleware(testHandler)

	// Test with X-Forwarded-For header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1, 127.0.0.1")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Request with X-Forwarded-For should succeed, got status %d", w.Code)
	}

	// Second request from same original IP should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1, 127.0.0.1")
	w2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status %d", w2.Code)
	}
}

func TestRateLimiterReset(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create rate limiter with 1 request per 100ms
	limiter := NewRateLimiter(1, 100*time.Millisecond)
	wrappedHandler := limiter.Middleware(testHandler)

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request should succeed, got status %d", w.Code)
	}

	// Second request from same IP should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	w2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status %d", w2.Code)
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Request after window should succeed from same IP
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	w3 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Request after window reset should succeed, got status %d", w3.Code)
	}
}

func TestRateLimiterMultipleClients(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create rate limiter with 1 request per second
	limiter := NewRateLimiter(1, time.Second)
	wrappedHandler := limiter.Middleware(testHandler)

	// First client request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	w1 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First client's request should succeed, got status %d", w1.Code)
	}

	// Second client request should succeed (different client)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second client's request should succeed, got status %d", w2.Code)
	}

	// Third request from first client should be rate limited
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	w3 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("First client's second request should be rate limited, got status %d", w3.Code)
	}
}
