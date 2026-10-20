package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/gorm"
)

const (
	dashboardCacheTTL    = 5 * time.Minute
	dashboardCachePrefix = "kor:dashboard:"
)

// AnalyticsService computes user-level analytics for the dashboard.
type AnalyticsService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewAnalyticsService creates an AnalyticsService.
func NewAnalyticsService(db *gorm.DB, rdb *redis.Client) *AnalyticsService {
	return &AnalyticsService{db: db, redis: rdb}
}

// PortfolioSummary contains a user's asset holdings overview.
type PortfolioSummary struct {
	TotalAssets       int64   `json:"total_assets"`
	TotalValueXLM     float64 `json:"total_value_xlm"`
	TotalTransactions int64   `json:"total_transactions"`
	UniqueAssetTypes  int64   `json:"unique_asset_types"`
}

// TransactionVolume holds daily transaction counts for charting.
type TransactionVolume struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// ActivitySummary is the full dashboard payload returned to the frontend.
type ActivitySummary struct {
	Portfolio         PortfolioSummary      `json:"portfolio"`
	RecentActivities  []models.UserActivity `json:"recent_activities"`
	TransactionVolume []TransactionVolume   `json:"transaction_volume_30d"`
	GeneratedAt       time.Time             `json:"generated_at"`
}

// GetDashboard returns the full activity dashboard for a user, backed by a 5-minute cache.
func (s *AnalyticsService) GetDashboard(ctx context.Context, userID uint) (*ActivitySummary, error) {
	cacheKey := fmt.Sprintf("%s%d", dashboardCachePrefix, userID)

	if s.redis != nil {
		if cached, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var summary ActivitySummary
			if json.Unmarshal(cached, &summary) == nil {
				return &summary, nil
			}
		}
	}

	summary, err := s.computeDashboard(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.redis != nil {
		if data, err := json.Marshal(summary); err == nil {
			s.redis.Set(ctx, cacheKey, data, dashboardCacheTTL)
		}
	}

	return summary, nil
}

func (s *AnalyticsService) computeDashboard(ctx context.Context, userID uint) (*ActivitySummary, error) {
	var portfolio PortfolioSummary

	s.db.WithContext(ctx).Model(&models.UserBalance{}).
		Where("user_id = ? AND balance > 0", userID).
		Count(&portfolio.TotalAssets)

	s.db.WithContext(ctx).Model(&models.Transaction{}).
		Joins("JOIN assets ON assets.id = transactions.asset_id").
		Where(`transactions.from_address = (SELECT stellar_address FROM users WHERE id = ?) OR
			transactions.to_address = (SELECT stellar_address FROM users WHERE id = ?)`, userID, userID).
		Count(&portfolio.TotalTransactions)

	type assetTypeCount struct {
		Count int64
	}
	var atc assetTypeCount
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT a.asset_type) AS count
		FROM user_balances ub
		JOIN assets a ON a.id = ub.asset_id
		WHERE ub.user_id = ? AND ub.balance > 0 AND a.deleted_at IS NULL`, userID).Scan(&atc)
	portfolio.UniqueAssetTypes = atc.Count

	var activities []models.UserActivity
	s.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&activities)

	since := time.Now().UTC().AddDate(0, 0, -30)
	var volume []TransactionVolume
	s.db.WithContext(ctx).Raw(`
		SELECT DATE(created_at) AS date, COUNT(*) AS count
		FROM user_activities
		WHERE user_id = ? AND created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date ASC`, userID, since).Scan(&volume)

	return &ActivitySummary{
		Portfolio:         portfolio,
		RecentActivities:  activities,
		TransactionVolume: volume,
		GeneratedAt:       time.Now().UTC(),
	}, nil
}

// RecordActivity appends a timestamped activity record for a user.
func (s *AnalyticsService) RecordActivity(ctx context.Context, userID uint, actType models.ActivityType, resourceID, resourceType, metadata, ip string) error {
	act := models.UserActivity{
		UserID:       userID,
		Type:         actType,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Metadata:     metadata,
		IPAddress:    ip,
	}
	if err := s.db.WithContext(ctx).Create(&act).Error; err != nil {
		return fmt.Errorf("record activity: %w", err)
	}

	if s.redis != nil {
		cacheKey := fmt.Sprintf("%s%d", dashboardCachePrefix, userID)
		s.redis.Del(ctx, cacheKey)
	}
	return nil
}

// GetActivityTimeline returns a user's paginated activity timeline.
func (s *AnalyticsService) GetActivityTimeline(ctx context.Context, userID uint, actType string, limit, offset int) ([]models.UserActivity, int64, error) {
	var activities []models.UserActivity
	var total int64

	q := s.db.WithContext(ctx).Where("user_id = ?", userID)
	if actType != "" {
		q = q.Where("type = ?", actType)
	}
	q.Model(&models.UserActivity{}).Count(&total)
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&activities).Error
	return activities, total, err
}

// ExportActivityReport returns all activity entries for a user, suitable for CSV export.
func (s *AnalyticsService) ExportActivityReport(ctx context.Context, userID uint, from, to time.Time) ([]models.UserActivity, error) {
	var activities []models.UserActivity
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, from, to).
		Order("created_at ASC").
		Find(&activities).Error
	return activities, err
}
