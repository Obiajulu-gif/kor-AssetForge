package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/gorm"
)

// RateLimitDashboardHandler exposes rate-limit metrics for the monitoring dashboard.
type RateLimitDashboardHandler struct {
	Redis *redis.Client
	DB    *gorm.DB
}

// NewRateLimitDashboardHandler creates a RateLimitDashboardHandler.
func NewRateLimitDashboardHandler(rdb *redis.Client, db *gorm.DB) *RateLimitDashboardHandler {
	return &RateLimitDashboardHandler{Redis: rdb, DB: db}
}

// RateLimitStats holds aggregated rate-limit hit data for a key.
type RateLimitStats struct {
	Key      string    `json:"key"`
	Hits     int64     `json:"hits"`
	LastSeen time.Time `json:"last_seen"`
}

// GetStats returns live rate-limit hit counts from Redis plus historical
// aggregations from the database (top violators, per-endpoint breakdown).
// GET /api/v1/admin/rate-limit/stats
func (h *RateLimitDashboardHandler) GetStats(c *gin.Context) {
	ctx := context.Background()
	pattern := "rate_limiter:*"

	keys, err := h.Redis.Keys(ctx, pattern).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch rate limit keys"})
		return
	}

	var liveStats []RateLimitStats
	for _, k := range keys {
		val, err := h.Redis.Get(ctx, k).Int64()
		if err != nil {
			continue
		}
		ttl, _ := h.Redis.TTL(ctx, k).Result()
		liveStats = append(liveStats, RateLimitStats{
			Key:      k,
			Hits:     val,
			LastSeen: time.Now().UTC().Add(-ttl),
		})
	}

	// Historical aggregation: top violators in last 24 h.
	var topViolators []models.RateLimitSummary
	if h.DB != nil {
		h.DB.Raw(`
			SELECT client_key, endpoint, COUNT(*) AS total_hits,
			       MIN(hit_at) AS first_seen, MAX(hit_at) AS last_seen
			FROM rate_limit_events
			WHERE hit_at >= NOW() - INTERVAL '24 hours'
			GROUP BY client_key, endpoint
			ORDER BY total_hits DESC
			LIMIT 50
		`).Scan(&topViolators)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"live_count":    len(liveStats),
		"live_data":     liveStats,
		"top_violators": topViolators,
	})
}

// GetHistoricalStats returns per-endpoint rate-limit hit counts over a time window.
// GET /api/v1/admin/rate-limit/history?hours=24&endpoint=/api/v1/assets
func (h *RateLimitDashboardHandler) GetHistoricalStats(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours < 1 || hours > 720 {
		hours = 24
	}
	endpoint := c.Query("endpoint") // optional filter

	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)

	type HourlyStat struct {
		Hour      string `json:"hour"`
		Endpoint  string `json:"endpoint"`
		TotalHits int64  `json:"total_hits"`
	}

	var rows []HourlyStat
	if err := h.DB.WithContext(c.Request.Context()).Raw(`
		SELECT DATE_TRUNC('hour', hit_at) AS hour, endpoint, COUNT(*) AS total_hits
		FROM rate_limit_events
		WHERE hit_at >= ? AND (? = '' OR endpoint = ?)
		GROUP BY 1, 2
		ORDER BY 1 ASC`, since, endpoint, endpoint).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch rate limit history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"hours":    hours,
		"endpoint": endpoint,
		"data":     rows,
	})
}

// RecordEvent persists a rate-limit violation event. Intended to be called
// from the rate-limit middleware (non-blocking, fire-and-forget).
func (h *RateLimitDashboardHandler) RecordEvent(clientKey, endpoint, method string) {
	if h.DB == nil {
		return
	}
	event := models.RateLimitEvent{
		ClientKey: clientKey,
		Endpoint:  endpoint,
		Method:    method,
		HitAt:     time.Now().UTC(),
	}
	h.DB.Create(&event)
}

// ResetKey clears rate-limit state for a specific key (admin action).
// DELETE /api/v1/admin/rate-limit/:key
func (h *RateLimitDashboardHandler) ResetKey(c *gin.Context) {
	key := "rate_limiter:" + c.Param("key")
	ctx := context.Background()
	h.Redis.Del(ctx, key)
	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": key})
}
