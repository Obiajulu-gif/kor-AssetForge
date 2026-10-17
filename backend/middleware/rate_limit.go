package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// rateLimitEntry pairs a token-bucket limiter with a last-seen timestamp used
// for idle eviction.
type rateLimitEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitConfig holds per-method rate limit parameters loaded from env.
type RateLimitConfig struct {
	// GetRPS is the sustained requests-per-second allowed for GET requests.
	GetRPS float64
	// GetBurst is the maximum burst size for GET requests.
	GetBurst int
	// MutateRPS is the sustained requests-per-second for POST/PUT/PATCH/DELETE.
	MutateRPS float64
	// MutateBurst is the maximum burst size for mutating requests.
	MutateBurst int
	// AdminIPs is the set of IP addresses that bypass rate limiting entirely.
	AdminIPs map[string]struct{}
	// logger is used to emit throttle warnings.
	logger *zap.SugaredLogger
}

// RateLimitConfigFromEnv reads configuration from environment variables:
//
//	RATE_LIMIT_GET_RPS       float, default 100
//	RATE_LIMIT_GET_BURST     int,   default 20
//	RATE_LIMIT_MUTATE_RPS    float, default 20
//	RATE_LIMIT_MUTATE_BURST  int,   default 10
//	RATE_LIMIT_ADMIN_IPS     comma-separated IPs, default empty
func RateLimitConfigFromEnv(logger *zap.SugaredLogger) *RateLimitConfig {
	cfg := &RateLimitConfig{
		GetRPS:      parseFloatEnv("RATE_LIMIT_GET_RPS", 100),
		GetBurst:    parseIntEnv("RATE_LIMIT_GET_BURST", 20),
		MutateRPS:   parseFloatEnv("RATE_LIMIT_MUTATE_RPS", 20),
		MutateBurst: parseIntEnv("RATE_LIMIT_MUTATE_BURST", 10),
		AdminIPs:    parseAdminIPs(os.Getenv("RATE_LIMIT_ADMIN_IPS")),
		logger:      logger,
	}
	return cfg
}

// perClientStore holds per-client limiters for a single rate tier.
type perClientStore struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	rps     float64
	burst   int
}

func newPerClientStore(rps float64, burst int) *perClientStore {
	s := &perClientStore{
		entries: make(map[string]*rateLimitEntry),
		rps:     rps,
		burst:   burst,
	}
	go s.cleanupLoop()
	return s
}

func (s *perClientStore) allow(key string) (bool, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entries[key]
	if !ok {
		e = &rateLimitEntry{
			limiter: rate.NewLimiter(rate.Limit(s.rps), s.burst),
		}
		s.entries[key] = e
	}
	e.lastSeen = time.Now()
	return e.limiter.Allow(), e.limiter.Tokens()
}

// cleanupLoop removes entries that have been idle for more than 10 minutes.
func (s *perClientStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		s.mu.Lock()
		for key, e := range s.entries {
			if e.lastSeen.Before(cutoff) {
				delete(s.entries, key)
			}
		}
		s.mu.Unlock()
	}
}

// AdvancedRateLimit returns a Gin middleware that enforces configurable per-IP
// (or per-JWT-user) rate limits with separate budgets for read vs. mutating
// requests and an admin-IP bypass list.
//
// The key used for bucketing is, in priority order:
//  1. JWT user ID stored in the gin context under key "user_id"
//  2. X-User-ID request header
//  3. Client IP
func AdvancedRateLimit(cfg *RateLimitConfig) gin.HandlerFunc {
	getStore := newPerClientStore(cfg.GetRPS, cfg.GetBurst)
	mutateStore := newPerClientStore(cfg.MutateRPS, cfg.MutateBurst)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Admin-IP exemption.
		if _, exempt := cfg.AdminIPs[ip]; exempt {
			c.Next()
			return
		}

		// Determine bucketing key (user-scoped > IP-scoped).
		key := ip
		if uid, ok := c.Get("user_id"); ok {
			key = fmt.Sprintf("uid:%v", uid)
		} else if header := c.GetHeader("X-User-ID"); header != "" {
			key = "uid:" + header
		}

		// Choose store based on HTTP method.
		isMutating := c.Request.Method != http.MethodGet &&
			c.Request.Method != http.MethodHead &&
			c.Request.Method != http.MethodOptions
		store := getStore
		if isMutating {
			store = mutateStore
		}

		allowed, tokens := store.allow(key)

		// Standard rate limit headers (issue #303), on every response from
		// this middleware so clients can implement backoff themselves.
		remaining := int(tokens)
		if remaining < 0 {
			remaining = 0
		}
		resetIn := 1
		if store.rps > 0 {
			resetIn = int(float64(store.burst) / store.rps)
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(store.burst))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Duration(resetIn)*time.Second).Unix(), 10))

		if !allowed {
			if cfg.logger != nil {
				cfg.logger.Warnw("rate limit exceeded",
					"key", key,
					"method", c.Request.Method,
					"path", c.FullPath(),
					"ip", ip,
				)
			}
			retryAfter := int(1.0 / store.rps)
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}

// --- helpers ----------------------------------------------------------------

func parseFloatEnv(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return fallback
}

func parseIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			return i
		}
	}
	return fallback
}

func parseAdminIPs(raw string) map[string]struct{} {
	m := make(map[string]struct{})
	if raw == "" {
		return m
	}
	for start, i := 0, 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ',' {
			ip := strings.TrimSpace(raw[start:i])
			if ip != "" {
				m[ip] = struct{}{}
			}
			start = i + 1
		}
	}
	return m
}
