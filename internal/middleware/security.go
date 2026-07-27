package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// Security Headers Middleware
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent Clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS Protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (Allowing Three.js, Swagger, HLS.js Workers, CDNs & WebSockets)
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' blob: https://cdnjs.cloudflare.com https://cdn.jsdelivr.net; "+
				"worker-src 'self' blob:; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com https://cdn.jsdelivr.net; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data: blob: https://validator.swagger.io; "+
				"media-src 'self' blob:; "+
				"connect-src 'self' ws: wss: https://cdn.jsdelivr.net https://cdnjs.cloudflare.com;")

		next.ServeHTTP(w, r)
	})
}

// IP Token-Bucket Rate Limiter Middleware for Brute-Force & DDoS Protection
type visitor struct {
	tokens     float64
	lastSeen   time.Time
	maxTokens  float64
	refillRate float64 // tokens per second
}

type IPRateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
}

func NewIPRateLimiter() *IPRateLimiter {
	limiter := &IPRateLimiter{
		visitors: make(map[string]*visitor),
	}

	// Cleanup inactive visitors every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.mu.Lock()
			for ip, v := range limiter.visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(limiter.visitors, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

func (lim *IPRateLimiter) RateLimit(maxTokens float64, refillRate float64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			lim.mu.Lock()
			v, exists := lim.visitors[ip]
			now := time.Now()

			if !exists {
				v = &visitor{
					tokens:     maxTokens - 1,
					lastSeen:   now,
					maxTokens:  maxTokens,
					refillRate: refillRate,
				}
				lim.visitors[ip] = v
				lim.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			// Refill tokens based on elapsed time
			elapsed := now.Sub(v.lastSeen).Seconds()
			v.tokens += elapsed * v.refillRate
			if v.tokens > v.maxTokens {
				v.tokens = v.maxTokens
			}
			v.lastSeen = now

			if v.tokens < 1 {
				lim.mu.Unlock()
				w.Header().Set("Retry-After", "5")
				http.Error(w, `{"error":"Demasiadas peticiones. Por favor espere unos segundos."}`, http.StatusTooManyRequests)
				return
			}

			v.tokens--
			lim.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
