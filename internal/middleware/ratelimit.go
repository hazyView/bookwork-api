package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"bookwork-api/internal/models"
)

// RateLimiter implements rate limiting with sliding window algorithm
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes old request entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for key, requests := range rl.requests {
			// Remove requests older than the window
			validRequests := make([]time.Time, 0, len(requests))
			for _, reqTime := range requests {
				if now.Sub(reqTime) < rl.window {
					validRequests = append(validRequests, reqTime)
				}
			}

			if len(validRequests) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validRequests
			}
		}
		rl.mutex.Unlock()
	}
}

// getClientKey extracts client identifier for rate limiting
func (rl *RateLimiter) getClientKey(r *http.Request) string {
	// Try to get user ID from Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Use a hash of the auth header as the key
		return fmt.Sprintf("auth_%s", authHeader[:min(10, len(authHeader))])
	}

	// Fallback to IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	return fmt.Sprintf("ip_%s", ip)
}

// isAllowed checks if the request is within rate limits
func (rl *RateLimiter) isAllowed(clientKey string) (bool, int, time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Get existing requests for this client
	requests, exists := rl.requests[clientKey]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// Remove requests older than the window
	validRequests := make([]time.Time, 0, len(requests))
	for _, reqTime := range requests {
		if now.Sub(reqTime) < rl.window {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we're within the limit
	if len(validRequests) >= rl.limit {
		// Calculate reset time (when the oldest request will expire)
		resetTime := validRequests[0].Add(rl.window)
		return false, rl.limit - len(validRequests), resetTime
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[clientKey] = validRequests

	return true, rl.limit - len(validRequests), now.Add(rl.window)
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientKey := rl.getClientKey(r)
		allowed, remaining, resetTime := rl.isAllowed(clientKey)

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, remaining)))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)

			response := &models.FrontendErrorResponse{
				Error:      "RateLimitExceeded",
				Message:    "Too many requests. Please wait before trying again.",
				StatusCode: http.StatusTooManyRequests,
				Details: map[string]interface{}{
					"limit":      rl.limit,
					"window":     rl.window.String(),
					"resetAt":    resetTime.Format(time.RFC3339),
					"retryAfter": int64(time.Until(resetTime).Seconds()),
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}

			// Use a simple JSON encoder since we can't import the full handlers package
			jsonResponse := fmt.Sprintf(`{"error":"%s","message":"%s","statusCode":%d,"timestamp":"%s","details":{"limit":%d,"retryAfter":%d}}`,
				response.Error, response.Message, response.StatusCode, response.Timestamp, rl.limit, int64(time.Until(resetTime).Seconds()))

			w.Write([]byte(jsonResponse))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
