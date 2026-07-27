package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handlerToTest := SecurityHeadersMiddleware(nextHandler)

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	rec := httptest.NewRecorder()

	handlerToTest.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("Missing or invalid X-Content-Type-Options header")
	}

	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("Missing or invalid X-Frame-Options header")
	}

	if rec.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("Missing or invalid X-XSS-Protection header")
	}
}

func TestIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter()
	rateLimitMW := limiter.RateLimit(2, 1) // 2 max tokens

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := rateLimitMW(nextHandler)

	// First request - should pass
	req1 := httptest.NewRequest("GET", "http://example.com/api/auth/login", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("Expected 200 OK on req1, got %d", rec1.Code)
	}

	// Second request - should pass
	rec2 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec2, req1)
	if rec2.Code != http.StatusOK {
		t.Errorf("Expected 200 OK on req2, got %d", rec2.Code)
	}

	// Third request - exceeded tokens -> should get 429 Too Many Requests
	rec3 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec3, req1)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 Too Many Requests on req3, got %d", rec3.Code)
	}
}
