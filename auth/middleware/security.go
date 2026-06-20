package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets defensive HTTP response headers on every response.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := newRequestID()
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// --- per-IP login rate limiter / lockout ---

const (
	maxLoginAttempts = 5
	lockoutDuration  = 15 * time.Minute
	attemptWindow    = 10 * time.Minute
)

type ipBucket struct {
	count    int
	lastFail time.Time
	lockedAt time.Time
}

var (
	bucketMu sync.Mutex
	buckets  = map[string]*ipBucket{}
)

// RateLimitLogin blocks IPs that have exceeded maxLoginAttempts within attemptWindow.
func RateLimitLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucketMu.Lock()
		b := getBucket(ip)
		if !b.lockedAt.IsZero() && time.Since(b.lockedAt) < lockoutDuration {
			bucketMu.Unlock()
			rid, _ := c.Get("request_id")
			slog.Warn("login_blocked",
				slog.String("event", "login"),
				slog.String("result", "locked"),
				slog.String("ip", ip),
				slog.Any("request_id", rid),
			)
			c.Header("Retry-After", "900")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many failed attempts, try again later"})
			c.Abort()
			return
		}
		bucketMu.Unlock()
		c.Next()
	}
}

// RecordLoginFailure increments the failure counter; locks the IP after maxLoginAttempts.
func RecordLoginFailure(ip string) {
	bucketMu.Lock()
	defer bucketMu.Unlock()
	b := getBucket(ip)
	if time.Since(b.lastFail) > attemptWindow {
		b.count = 0
		b.lockedAt = time.Time{}
	}
	b.count++
	b.lastFail = time.Now()
	if b.count >= maxLoginAttempts {
		b.lockedAt = time.Now()
	}
}

// RecordLoginSuccess resets the failure counter for the IP.
func RecordLoginSuccess(ip string) {
	bucketMu.Lock()
	defer bucketMu.Unlock()
	delete(buckets, ip)
}

func getBucket(ip string) *ipBucket {
	if b, ok := buckets[ip]; ok {
		return b
	}
	b := &ipBucket{}
	buckets[ip] = b
	return b
}
