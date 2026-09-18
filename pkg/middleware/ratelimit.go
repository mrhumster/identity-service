package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/pkg/dto"
)

// rateLimiter is a dependency-free in-memory fixed-window limiter. It is
// coarse by design: per-process, best-effort, enough to stop trivial burst
// flooding of auth endpoints (login/refresh/verify/resend) without extra
// storage.
type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]*windowCounter
}

type windowCounter struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:    limit,
		window:   window,
		counters: make(map[string]*windowCounter),
	}
}

func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	wc, ok := r.counters[key]
	if !ok || now.After(wc.resetAt) {
		// Sliding start: each new window resets the counter.
		r.counters[key] = &windowCounter{count: 1, resetAt: now.Add(r.window)}
		// Opportunistic cleanup to keep the map bounded (windows are short).
		if len(r.counters) > 10_000 {
			for k, c := range r.counters {
				if now.After(c.resetAt) {
					delete(r.counters, k)
				}
			}
		}
		return true
	}
	wc.count++
	return wc.count <= r.limit
}

// RateLimitPerMin rejects requests once a single caller exceeds limit within
// a minute. Authenticated callers are keyed by the tamper-proof user id that
// AuthMiddleware placed in the context; guests are keyed by client IP.
func RateLimitPerMin(limit int) gin.HandlerFunc {
	if limit <= 0 {
		limit = 30
	}
	rl := newRateLimiter(limit, time.Minute)
	return func(c *gin.Context) {
		var key string
		if user, ok := c.Get("user"); ok {
			if uid, ok := user.(uuid.UUID); ok && uid != uuid.Nil {
				key = "u:" + uid.String()
			}
		}
		if key == "" {
			key = "ip:" + c.ClientIP()
		}
		if !rl.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, dto.ErrorResponse("too many requests"))
			return
		}
		c.Next()
	}
}
