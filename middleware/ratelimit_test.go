package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(5), 10)

	if rl.r != rate.Limit(5) {
		t.Errorf("Expected rate 5, got %v", rl.r)
	}

	if rl.b != 10 {
		t.Errorf("Expected burst 10, got %d", rl.b)
	}

	if rl.limiters == nil {
		t.Error("Expected limiters map to be initialized")
	}
}

func TestGetLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(5), 10)

	// Get limiter for first IP
	limiter1 := rl.getLimiter("192.168.1.1")
	if limiter1 == nil {
		t.Error("Expected non-nil limiter")
	}

	// Get same IP again - should return same limiter
	limiter2 := rl.getLimiter("192.168.1.1")
	if limiter1 != limiter2 {
		t.Error("Expected same limiter instance for same IP")
	}

	// Get different IP - should return different limiter
	limiter3 := rl.getLimiter("192.168.1.2")
	if limiter1 == limiter3 {
		t.Error("Expected different limiter instance for different IP")
	}
}

func TestRateLimit(t *testing.T) {
	// Very low rate for testing - 1 request per second, burst of 2
	rl := NewRateLimiter(rate.Limit(1), 2)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	wrapped := rl.RateLimit(testHandler)

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345" //nolint:goconst
	rr1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("First request: expected status 200, got %d", rr1.Code)
	}

	// Second request should succeed (within burst)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rr2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Second request: expected status 200, got %d", rr2.Code)
	}

	// Third request should be rate limited
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	rr3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("Third request: expected status 429, got %d", rr3.Code)
	}

	// Different IP should not be affected
	req4 := httptest.NewRequest("GET", "/test", nil)
	req4.RemoteAddr = "192.168.1.2:12345"
	rr4 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr4, req4)

	if rr4.Code != http.StatusOK {
		t.Errorf("Request from different IP: expected status 200, got %d", rr4.Code)
	}
}

func TestGetIP(t *testing.T) {
	tests := []struct {
		name        string
		remoteAddr  string
		xForwardFor string
		xRealIP     string
		expected    string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1:12345",
		},
		{
			name:        "X-Forwarded-For header",
			remoteAddr:  "10.0.0.1:12345",
			xForwardFor: "203.0.113.1",
			expected:    "203.0.113.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.2",
			expected:   "203.0.113.2",
		},
		{
			name:        "X-Forwarded-For takes precedence",
			remoteAddr:  "10.0.0.1:12345",
			xForwardFor: "203.0.113.1",
			xRealIP:     "203.0.113.2",
			expected:    "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := getIP(req)
			if got != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	// This is a basic test - in reality, cleanup runs in background
	rl := NewRateLimiter(rate.Limit(100), 5)

	// Add some limiters
	rl.getLimiter("192.168.1.1")
	rl.getLimiter("192.168.1.2")
	rl.getLimiter("192.168.1.3")

	if len(rl.limiters) != 3 {
		t.Errorf("Expected 3 limiters, got %d", len(rl.limiters))
	}

	// Wait a bit and check cleanup doesn't break anything
	time.Sleep(100 * time.Millisecond)

	// Limiters should still work
	limiter := rl.getLimiter("192.168.1.1")
	if limiter == nil {
		t.Error("Expected limiter to still exist")
	}
}
