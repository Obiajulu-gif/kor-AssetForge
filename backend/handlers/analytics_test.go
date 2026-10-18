package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserGrowthRejectsSQLInputAndReturnsDailyCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&models.User{}))
	for i, created := range []time.Time{time.Now().UTC(), time.Now().UTC().AddDate(0, 0, -40)} {
		require.NoError(t, db.Create(&models.User{
			StellarAddress: []string{"recent-address", "old-address"}[i],
			Email:          []string{"recent@example.com", "old@example.com"}[i],
			Username:       []string{"recent", "old"}[i], CreatedAt: created,
		}).Error)
	}
	router := gin.New()
	router.GET("/growth", NewAnalyticsHandler(db, nil).GetUserGrowth)
	for _, days := range []string{"30; DROP TABLE users;--", "0", "-1", "366", "abc"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/growth?days="+url.QueryEscape(days), nil))
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/growth?days=30", nil))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var result struct {
		Days int `json:"days"`
		Data []struct {
			Count int64 `json:"count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Equal(t, 30, result.Days)
	require.Len(t, result.Data, 1)
	require.EqualValues(t, 1, result.Data[0].Count)
	var count int64
	require.NoError(t, db.Model(&models.User{}).Count(&count).Error)
	require.EqualValues(t, 2, count)
}
