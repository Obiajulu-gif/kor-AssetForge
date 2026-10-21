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
	latestValuationCachePrefix = "kor:valuation:latest:"
	latestValuationCacheTTL    = 5 * time.Minute
)

// ValuationTrackerService records and queries asset valuation history.
type ValuationTrackerService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewValuationTrackerService creates a ValuationTrackerService.
func NewValuationTrackerService(db *gorm.DB, rdb *redis.Client) *ValuationTrackerService {
	return &ValuationTrackerService{db: db, redis: rdb}
}

// RecordValuation persists a new valuation snapshot for the given asset.
// It also invalidates the cached "latest" entry for that asset.
func (s *ValuationTrackerService) RecordValuation(assetID uint, valuationUSD float64, currency, source, notes string) (*models.ValuationHistory, error) {
	v := &models.ValuationHistory{
		AssetID:      assetID,
		ValuationUSD: valuationUSD,
		Currency:     currency,
		Source:       source,
		Notes:        notes,
		RecordedAt:   time.Now().UTC(),
	}
	if err := s.db.Create(v).Error; err != nil {
		return nil, fmt.Errorf("record valuation: %w", err)
	}

	if s.redis != nil {
		s.redis.Del(context.Background(), fmt.Sprintf("%s%d", latestValuationCachePrefix, assetID))
	}

	return v, nil
}

// GetHistory returns all valuation snapshots for an asset ordered by RecordedAt desc.
func (s *ValuationTrackerService) GetHistory(assetID uint, limit, offset int) ([]models.ValuationHistory, int64, error) {
	var records []models.ValuationHistory
	var total int64

	base := s.db.Model(&models.ValuationHistory{}).Where("asset_id = ?", assetID)
	base.Count(&total)
	err := base.Order("recorded_at DESC").Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

// GetTrend returns daily aggregated valuation data for the given asset and date range.
// granularity must be "daily", "weekly", or "monthly".
func (s *ValuationTrackerService) GetTrend(assetID uint, from, to time.Time, granularity string) ([]models.ValuationTrend, error) {
	var truncExpr string
	switch granularity {
	case "weekly":
		truncExpr = "week"
	case "monthly":
		truncExpr = "month"
	default:
		truncExpr = "day"
	}

	var rows []models.ValuationTrend
	err := s.db.Raw(`
		SELECT
			DATE_TRUNC(CAST(? AS text), recorded_at)::date::text AS date,
			AVG(valuation_usd)                         AS avg_valuation,
			MIN(valuation_usd)                         AS min_valuation,
			MAX(valuation_usd)                         AS max_valuation,
			COUNT(*)                                   AS snapshots
		FROM valuation_histories
		WHERE asset_id = ? AND recorded_at BETWEEN ? AND ? AND deleted_at IS NULL
		GROUP BY 1
		ORDER BY 1 ASC
	`, truncExpr, assetID, from, to).Scan(&rows).Error

	return rows, err
}

// GetLatestValuation returns the most recent valuation for an asset, using Redis
// as a short-lived cache to avoid repeated DB hits for frequently queried assets.
func (s *ValuationTrackerService) GetLatestValuation(ctx context.Context, assetID uint) (*models.ValuationHistory, error) {
	cacheKey := fmt.Sprintf("%s%d", latestValuationCachePrefix, assetID)

	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var v models.ValuationHistory
			if json.Unmarshal(raw, &v) == nil {
				return &v, nil
			}
		}
	}

	var v models.ValuationHistory
	if err := s.db.
		Where("asset_id = ?", assetID).
		Order("recorded_at DESC").
		First(&v).Error; err != nil {
		return nil, err
	}

	if s.redis != nil {
		if data, _ := json.Marshal(v); data != nil {
			s.redis.Set(ctx, cacheKey, data, latestValuationCacheTTL)
		}
	}

	return &v, nil
}

// RecordSaleValuation is a convenience wrapper called when a transaction completes,
// recording the sale price as a valuation event.
func (s *ValuationTrackerService) RecordSaleValuation(assetID uint, salePrice float64, currency string) error {
	_, err := s.RecordValuation(assetID, salePrice, currency, "sale", "recorded from completed transaction")
	return err
}
