package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// testPaymentUserID is the authenticated user injected into the gin context by
// the test router middleware.
const testPaymentUserID uint = 42

func setupPaymentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// A per-test in-memory database avoids state leaking between tests while
	// still allowing the connection pool to share a single database.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		// The payments table references users/assets which are irrelevant to
		// these unit tests, so skip constraint creation.
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Payment{}, &models.PaymentReconciliation{}))
	return db
}

func setupPaymentRouter(db *gorm.DB, userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewPaymentHandler(db)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})

	payments := r.Group("/api/v1/payments")
	{
		payments.POST("", h.CreatePayment)
		payments.GET("", h.ListPayments)
		payments.GET("/:id", h.GetPayment)
	}

	return r
}

func validPaymentBody() map[string]interface{} {
	return map[string]interface{}{
		"asset_id":    7,
		"amount_fiat": 15000,
		"currency":    "usd",
		"gateway":     "stripe",
	}
}

func createPayment(
	t *testing.T,
	r *gin.Engine,
	body map[string]interface{},
	idempotencyKey string,
) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func countPayments(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&models.Payment{}).Count(&count).Error)
	return count
}

// ---------------------------------------------------------------------------
// Idempotency key requirement
// ---------------------------------------------------------------------------

func TestCreatePaymentRequiresIdempotencyKey(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	w := createPayment(t, r, validPaymentBody(), "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, int64(0), countPayments(t, db))
}

func TestCreatePaymentRejectsTooShortIdempotencyKey(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	w := createPayment(t, r, validPaymentBody(), "short")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, int64(0), countPayments(t, db))
}

func TestCreatePaymentAcceptsIdempotencyKeyInBody(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	body := validPaymentBody()
	body["idempotency_key"] = "body-key-0001"

	w := createPayment(t, r, body, "")

	require.Equal(t, http.StatusCreated, w.Code)
	resp := decodeBody(t, w)
	assert.Equal(t, "body-key-0001", resp["idempotency_key"])
	assert.Equal(t, int64(1), countPayments(t, db))
}

// ---------------------------------------------------------------------------
// Happy path
// ---------------------------------------------------------------------------

func TestCreatePaymentPersistsIdempotencyKey(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	w := createPayment(t, r, validPaymentBody(), "idem-create-0001")
	require.Equal(t, http.StatusCreated, w.Code)

	resp := decodeBody(t, w)
	assert.NotZero(t, resp["payment_id"])
	assert.Equal(t, false, resp["idempotent_replay"])
	assert.Equal(t, "idem-create-0001", resp["idempotency_key"])
	assert.Equal(t, "pending", resp["status"])

	var payment models.Payment
	require.NoError(t, db.First(&payment).Error)
	assert.Equal(t, "idem-create-0001", payment.IdempotencyKey)
	assert.Equal(t, "USD", payment.Currency) // handler normalises currency
	assert.Equal(t, models.PaymentStatusPending, payment.Status)
}

// ---------------------------------------------------------------------------
// Duplicate requests return the same result
// ---------------------------------------------------------------------------

func TestCreatePaymentRetryReturnsSameResult(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	first := createPayment(t, r, validPaymentBody(), "idem-retry-0001")
	require.Equal(t, http.StatusCreated, first.Code)

	second := createPayment(t, r, validPaymentBody(), "idem-retry-0001")
	require.Equal(t, http.StatusOK, second.Code, "replayed request should not create a second payment")

	firstResp := decodeBody(t, first)
	secondResp := decodeBody(t, second)

	assert.Equal(t, firstResp["payment_id"], secondResp["payment_id"])
	assert.Equal(t, firstResp["gateway_payment_id"], secondResp["gateway_payment_id"])
	assert.Equal(t, firstResp["checkout_url"], secondResp["checkout_url"])
	assert.Equal(t, false, firstResp["idempotent_replay"])
	assert.Equal(t, true, secondResp["idempotent_replay"])

	assert.Equal(t, int64(1), countPayments(t, db))
}

func TestCreatePaymentDifferentKeysCreateDistinctPayments(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	first := createPayment(t, r, validPaymentBody(), "idem-distinct-01")
	require.Equal(t, http.StatusCreated, first.Code)
	second := createPayment(t, r, validPaymentBody(), "idem-distinct-02")
	require.Equal(t, http.StatusCreated, second.Code)

	assert.NotEqual(t,
		decodeBody(t, first)["payment_id"],
		decodeBody(t, second)["payment_id"],
	)
	assert.Equal(t, int64(2), countPayments(t, db))
}

// ---------------------------------------------------------------------------
// Reused keys with different parameters are rejected
// ---------------------------------------------------------------------------

func TestCreatePaymentRejectsSameKeyDifferentParams(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	first := createPayment(t, r, validPaymentBody(), "idem-conflict-01")
	require.Equal(t, http.StatusCreated, first.Code)

	tampered := validPaymentBody()
	tampered["amount_fiat"] = 99000

	second := createPayment(t, r, tampered, "idem-conflict-01")
	assert.Equal(t, http.StatusConflict, second.Code)
	assert.Equal(t, int64(1), countPayments(t, db))
}

func TestCreatePaymentConflictOnGatewayChange(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	first := createPayment(t, r, validPaymentBody(), "idem-conflict-02")
	require.Equal(t, http.StatusCreated, first.Code)

	otherGateway := validPaymentBody()
	otherGateway["gateway"] = "paypal"

	second := createPayment(t, r, otherGateway, "idem-conflict-02")
	assert.Equal(t, http.StatusConflict, second.Code)
	assert.Equal(t, int64(1), countPayments(t, db))
}

// ---------------------------------------------------------------------------
// Keys expire after 24 hours
// ---------------------------------------------------------------------------

func TestIdempotencyKeyExpiresAfter24Hours(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	first := createPayment(t, r, validPaymentBody(), "idem-expiry-0001")
	require.Equal(t, http.StatusCreated, first.Code)
	firstResp := decodeBody(t, first)

	// Backdate the original payment beyond the 24h TTL.
	expired := time.Now().Add(-25 * time.Hour)
	require.NoError(t, db.Model(&models.Payment{}).
		Where("idempotency_key = ?", "idem-expiry-0001").
		Update("created_at", expired).Error)

	second := createPayment(t, r, validPaymentBody(), "idem-expiry-0001")
	require.Equal(t, http.StatusCreated, second.Code, "expired key should be reusable")
	secondResp := decodeBody(t, second)

	assert.NotEqual(t, firstResp["payment_id"], secondResp["payment_id"])
	assert.Equal(t, false, secondResp["idempotent_replay"])
	assert.Equal(t, int64(2), countPayments(t, db))
}

func TestIdempotencyKeyStillActiveJustUnder24Hours(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	first := createPayment(t, r, validPaymentBody(), "idem-active-0001")
	require.Equal(t, http.StatusCreated, first.Code)

	recent := time.Now().Add(-23 * time.Hour)
	require.NoError(t, db.Model(&models.Payment{}).
		Where("idempotency_key = ?", "idem-active-0001").
		Update("created_at", recent).Error)

	second := createPayment(t, r, validPaymentBody(), "idem-active-0001")
	require.Equal(t, http.StatusOK, second.Code)
	assert.Equal(t, true, decodeBody(t, second)["idempotent_replay"])
	assert.Equal(t, int64(1), countPayments(t, db))
}

// ---------------------------------------------------------------------------
// Idempotency is scoped per user
// ---------------------------------------------------------------------------

func TestIdempotencyKeyScopedPerUser(t *testing.T) {
	db := setupPaymentTestDB(t)
	routerA := setupPaymentRouter(db, 1)
	routerB := setupPaymentRouter(db, 2)

	a := createPayment(t, routerA, validPaymentBody(), "shared-key-0001")
	require.Equal(t, http.StatusCreated, a.Code)
	b := createPayment(t, routerB, validPaymentBody(), "shared-key-0001")
	require.Equal(t, http.StatusCreated, b.Code, "same key from another user must not replay")

	assert.NotEqual(t, decodeBody(t, a)["payment_id"], decodeBody(t, b)["payment_id"])
	assert.Equal(t, int64(2), countPayments(t, db))
}

// ---------------------------------------------------------------------------
// Error mapping and access control
// ---------------------------------------------------------------------------

func TestCreatePaymentRejectsUnsupportedGateway(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	body := validPaymentBody()
	body["gateway"] = "cash"
	w := createPayment(t, r, body, "idem-badgw-0001")

	// Binding validation rejects unknown gateways before the handler runs.
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, int64(0), countPayments(t, db))
}

func TestGetPaymentForbiddenForOtherUser(t *testing.T) {
	db := setupPaymentTestDB(t)
	ownerRouter := setupPaymentRouter(db, testPaymentUserID)

	created := createPayment(t, ownerRouter, validPaymentBody(), "idem-access-0001")
	require.Equal(t, http.StatusCreated, created.Code)
	paymentID := decodeBody(t, created)["payment_id"]

	otherRouter := setupPaymentRouter(db, 999)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/payments/%.0f", paymentID), nil)
	w := httptest.NewRecorder()
	otherRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestListPaymentsReturnsUserPayments(t *testing.T) {
	db := setupPaymentTestDB(t)
	r := setupPaymentRouter(db, testPaymentUserID)

	require.Equal(t, http.StatusCreated, createPayment(t, r, validPaymentBody(), "idem-list-0001").Code)
	require.Equal(t, http.StatusCreated, createPayment(t, r, validPaymentBody(), "idem-list-0002").Code)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments?page=1&limit=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []models.Payment `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Data, 2)
}
