package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kor-assetforge/models"
	"github.com/yourusername/kor-assetforge/services"
	"gorm.io/gorm"
)

// UserDashboardHandler exposes user activity analytics endpoints.
type UserDashboardHandler struct {
	DB        *gorm.DB
	Analytics *services.AnalyticsService
}

// NewUserDashboardHandler creates a UserDashboardHandler.
func NewUserDashboardHandler(db *gorm.DB, analytics *services.AnalyticsService) *UserDashboardHandler {
	return &UserDashboardHandler{DB: db, Analytics: analytics}
}

// GetDashboard returns the full activity dashboard for the authenticated user.
// GET /api/v1/dashboard
func (h *UserDashboardHandler) GetDashboard(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.Analytics.GetDashboard(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

// GetActivityTimeline returns a paginated activity timeline for the authenticated user.
// GET /api/v1/dashboard/activity?type=transfer&page=1&limit=20
func (h *UserDashboardHandler) GetActivityTimeline(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	actType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	limit = clampPageLimit(limit)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	activities, total, err := h.Analytics.GetActivityTimeline(c.Request.Context(), userID, actType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    activities,
		"meta": gin.H{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ExportReport streams a CSV of the user's activity for the requested date range.
// GET /api/v1/dashboard/export?from=2024-01-01&to=2024-12-31
func (h *UserDashboardHandler) ExportReport(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fromStr := c.DefaultQuery("from", time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02"))
	toStr := c.DefaultQuery("to", time.Now().UTC().Format("2006-01-02"))

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date (use YYYY-MM-DD)"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date (use YYYY-MM-DD)"})
		return
	}
	to = to.Add(24*time.Hour - time.Second)

	activities, err := h.Analytics.ExportActivityReport(c.Request.Context(), userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=activity_report.csv")

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	_ = w.Write([]string{"id", "type", "resource_id", "resource_type", "metadata", "ip_address", "created_at"})
	for _, a := range activities {
		_ = w.Write([]string{
			strconv.FormatUint(uint64(a.ID), 10),
			string(a.Type),
			a.ResourceID,
			a.ResourceType,
			a.Metadata,
			a.IPAddress,
			a.CreatedAt.Format(time.RFC3339),
		})
	}
}

// RecordActivity records a manual activity event for the authenticated user.
// POST /api/v1/dashboard/activity
func (h *UserDashboardHandler) RecordActivity(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Type         models.ActivityType `json:"type" binding:"required"`
		ResourceID   string              `json:"resource_id"`
		ResourceType string              `json:"resource_type"`
		Metadata     string              `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Analytics.RecordActivity(
		c.Request.Context(), userID, req.Type,
		req.ResourceID, req.ResourceType, req.Metadata, c.ClientIP(),
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record activity"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true})
}
