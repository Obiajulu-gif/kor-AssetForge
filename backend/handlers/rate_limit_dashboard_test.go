package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRateLimitHistoryBindsEndpointFilter(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&models.RateLimitEvent{}))
	attack := "' OR 1=1; DROP TABLE rate_limit_events; --"
	for _, endpoint := range []string{"/assets", "/users", attack} {
		require.NoError(t, db.Create(&models.RateLimitEvent{ClientKey: "test", Endpoint: endpoint, Method: "GET", HitAt: time.Now().UTC()}).Error)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/history", NewRateLimitDashboardHandler(nil, db).GetHistoricalStats)
	for _, endpoint := range []string{"", "/assets", attack, "missing"} {
		t.Run(endpoint, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/history?endpoint="+url.QueryEscape(endpoint), nil))
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var result struct {
				Data []struct {
					Endpoint  string `json:"endpoint"`
					TotalHits int64  `json:"total_hits"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			switch endpoint {
			case "":
				require.Len(t, result.Data, 3)
			case "missing":
				require.Empty(t, result.Data)
			default:
				require.Len(t, result.Data, 1)
				require.Equal(t, endpoint, result.Data[0].Endpoint)
				require.EqualValues(t, 1, result.Data[0].TotalHits)
			}
		})
	}
	var count int64
	require.NoError(t, db.Model(&models.RateLimitEvent{}).Count(&count).Error)
	require.EqualValues(t, 3, count)
}
