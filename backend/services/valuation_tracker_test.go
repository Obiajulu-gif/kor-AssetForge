package services

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Exercise PostgreSQL's actual DATE_TRUNC bind behavior, not a rendered SQL string.
func TestValuationTrendParameterizedGranularity(t *testing.T) {
	dsn := os.Getenv("SQL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SQL_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	db = db.Begin()
	require.NoError(t, db.Error)
	t.Cleanup(func() { db.Rollback() })
	require.NoError(t, db.AutoMigrate(&models.ValuationHistory{}))
	asset := models.Asset{Name: "SQL test", Symbol: "SQLTEST", AssetType: "art", TotalSupply: 1000, OwnerAddress: "owner", ContractID: "sql-test"}
	require.NoError(t, db.Create(&asset).Error)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	for i, day := range []int{1, 2, 12} {
		require.NoError(t, db.Create(&models.ValuationHistory{
			AssetID: asset.ID, ValuationUSD: float64(100 + i*200), Currency: "USD", Source: "manual",
			RecordedAt: time.Date(2026, 1, day, 12, 0, 0, 0, time.UTC),
		}).Error)
	}
	service := NewValuationTrackerService(db, nil)
	for _, tc := range []struct {
		granularity string
		buckets     int
	}{
		{"daily", 3}, {"weekly", 2}, {"monthly", 1},
		{"day'); DROP TABLE valuation_histories; --", 3},
	} {
		t.Run(tc.granularity, func(t *testing.T) {
			trend, err := service.GetTrend(asset.ID, from, to, tc.granularity)
			require.NoError(t, err)
			require.Len(t, trend, tc.buckets)
			var snapshots int64
			for _, bucket := range trend {
				snapshots += bucket.Snapshots
			}
			require.EqualValues(t, 3, snapshots)
			if tc.granularity == "monthly" {
				require.Equal(t, float64(300), trend[0].AvgValuation)
			}
		})
	}
	var count int64
	require.NoError(t, db.Model(&models.ValuationHistory{}).Count(&count).Error)
	require.EqualValues(t, 3, count)
}
