package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/gorm"
)

const maxWatchlistItemsPerList = 100

// WatchlistHandler manages watchlist CRUD and asset membership endpoints.
type WatchlistHandler struct {
	DB *gorm.DB
}

// NewWatchlistHandler creates a WatchlistHandler.
func NewWatchlistHandler(db *gorm.DB) *WatchlistHandler {
	return &WatchlistHandler{DB: db}
}

func currentUserID(c *gin.Context) uint {
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// ListWatchlists returns all watchlists owned by the authenticated user.
// GET /api/v1/watchlists
func (h *WatchlistHandler) ListWatchlists(c *gin.Context) {
	userID := currentUserID(c)
	var lists []models.Watchlist
	if err := h.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		Find(&lists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch watchlists"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": lists})
}

// CreateWatchlist creates a new watchlist for the authenticated user.
// POST /api/v1/watchlists
func (h *WatchlistHandler) CreateWatchlist(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,min=1,max=100"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wl := models.Watchlist{
		UserID:      currentUserID(c),
		Name:        req.Name,
		Description: req.Description,
		IsPublic:    req.IsPublic,
	}
	if err := h.DB.Create(&wl).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create watchlist"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": wl})
}

// GetWatchlist returns a watchlist with its items. Public lists are visible to all.
// GET /api/v1/watchlists/:id
func (h *WatchlistHandler) GetWatchlist(c *gin.Context) {
	var wl models.Watchlist
	if err := h.DB.Where("deleted_at IS NULL").
		Preload("Items.Asset").
		First(&wl, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "watchlist not found"})
		return
	}

	userID := currentUserID(c)
	if !wl.IsPublic && wl.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": wl})
}

// UpdateWatchlist updates name, description or visibility.
// PATCH /api/v1/watchlists/:id
func (h *WatchlistHandler) UpdateWatchlist(c *gin.Context) {
	var wl models.Watchlist
	if err := h.DB.Where("deleted_at IS NULL AND user_id = ?", currentUserID(c)).
		First(&wl, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "watchlist not found"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		wl.Name = *req.Name
	}
	if req.Description != nil {
		wl.Description = *req.Description
	}
	if req.IsPublic != nil {
		wl.IsPublic = *req.IsPublic
	}

	if err := h.DB.Save(&wl).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update watchlist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": wl})
}

// DeleteWatchlist soft-deletes a watchlist owned by the authenticated user.
// DELETE /api/v1/watchlists/:id
func (h *WatchlistHandler) DeleteWatchlist(c *gin.Context) {
	result := h.DB.Where("user_id = ?", currentUserID(c)).Delete(&models.Watchlist{}, c.Param("id"))
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "watchlist not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AddAsset adds an asset to a watchlist (max 100 items per list).
// POST /api/v1/watchlists/:id/assets
func (h *WatchlistHandler) AddAsset(c *gin.Context) {
	var wl models.Watchlist
	if err := h.DB.Where("user_id = ? AND deleted_at IS NULL", currentUserID(c)).
		First(&wl, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "watchlist not found"})
		return
	}

	var req struct {
		AssetID    uint     `json:"asset_id" binding:"required"`
		Notes      string   `json:"notes"`
		AlertPrice *float64 `json:"alert_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int64
	h.DB.Model(&models.WatchlistItem{}).Where("watchlist_id = ?", wl.ID).Count(&count)
	if count >= maxWatchlistItemsPerList {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "watchlist is full (max 100 assets)"})
		return
	}

	item := models.WatchlistItem{
		WatchlistID: wl.ID,
		AssetID:     req.AssetID,
		Notes:       req.Notes,
		AlertPrice:  req.AlertPrice,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add asset (it may already be in this list)"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": item})
}

// RemoveAsset removes an asset from a watchlist.
// DELETE /api/v1/watchlists/:id/assets/:assetId
func (h *WatchlistHandler) RemoveAsset(c *gin.Context) {
	var wl models.Watchlist
	if err := h.DB.Where("user_id = ? AND deleted_at IS NULL", currentUserID(c)).
		First(&wl, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "watchlist not found"})
		return
	}

	assetID, err := strconv.ParseUint(c.Param("assetId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	result := h.DB.Where("watchlist_id = ? AND asset_id = ?", wl.ID, assetID).
		Delete(&models.WatchlistItem{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not in watchlist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListPublicWatchlists returns all publicly shared watchlists.
// GET /api/v1/watchlists/public
func (h *WatchlistHandler) ListPublicWatchlists(c *gin.Context) {
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

	var lists []models.Watchlist
	var total int64
	base := h.DB.Model(&models.Watchlist{}).Where("is_public = true AND deleted_at IS NULL")
	base.Count(&total)
	base.Offset(offset).Limit(limit).Find(&lists)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    lists,
		"meta": gin.H{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
