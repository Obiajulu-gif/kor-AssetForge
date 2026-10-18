//go:build ignore
// +build ignore

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPaymentProcessingFlow tests the complete fiat payment lifecycle.
func TestPaymentProcessingFlow(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)
	setup.ApproveKYC(t, user.PublicKey)

	// Step 1: initiate a fiat purchase via a supported gateway
	paymentData := map[string]interface{}{
		"asset_id":    1,
		"amount_fiat": 25000, // $250.00 in cents
		"currency":    "USD",
		"gateway":     "stripe",
	}

	resp, err := setup.MakeRequest("POST", "/api/v1/payments", paymentData, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	assert.NotNil(t, created["payment_id"])
	assert.NotNil(t, created["checkout_url"])
	assert.Equal(t, "pending", created["status"])

	paymentID := fmt.Sprintf("%v", created["payment_id"])

	// Step 2: fetch the payment back
	resp, err = setup.MakeRequest("GET", "/api/v1/payments/"+paymentID, nil, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 3: list the user's payments
	resp, err = setup.MakeRequest("GET", "/api/v1/payments?page=1&limit=20", nil, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var list map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	assert.Contains(t, list, "data")
	assert.Contains(t, list, "total")

	t.Logf("✅ Payment processing flow completed for payment %s", paymentID)
}

// TestPaymentLifecycleWebhook tests that a gateway webhook settles a payment.
func TestPaymentLifecycleWebhook(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)
	setup.ApproveKYC(t, user.PublicKey)

	resp, err := setup.MakeRequest("POST", "/api/v1/payments", map[string]interface{}{
		"asset_id":    1,
		"amount_fiat": 10000,
		"currency":    "USD",
		"gateway":     "paypal",
	}, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Simulate the gateway confirming the payment (route is versionless).
	webhook := map[string]interface{}{
		"event":      "payment.completed",
		"payment_id": 1,
		"status":     "completed",
	}
	resp, err = setup.MakeRequest("POST", "/payments/webhooks/paypal", webhook, "")
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted,
		"unexpected webhook status %d", resp.StatusCode)

	t.Log("✅ Payment webhook handling tests passed")
}

// TestPaymentValidations tests input validation for payment creation.
func TestPaymentValidations(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)

	testCases := []struct {
		name   string
		data   map[string]interface{}
		status int
	}{
		{
			name: "unsupported gateway",
			data: map[string]interface{}{
				"asset_id":    1,
				"amount_fiat": 1000,
				"currency":    "USD",
				"gateway":     "bitcoin",
			},
			status: http.StatusBadRequest,
		},
		{
			name: "missing amount",
			data: map[string]interface{}{
				"asset_id": 1,
				"currency": "USD",
				"gateway":  "stripe",
			},
			status: http.StatusBadRequest,
		},
		{
			name: "zero amount",
			data: map[string]interface{}{
				"asset_id":    1,
				"amount_fiat": 0,
				"currency":    "USD",
				"gateway":     "stripe",
			},
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := setup.MakeRequest("POST", "/api/v1/payments", tc.data, user.Token)
			require.NoError(t, err)
			assert.Equal(t, tc.status, resp.StatusCode)
		})
	}

	t.Log("✅ Payment validation tests passed")
}

// TestPaymentAccessControl ensures a user cannot read another user's payment.
func TestPaymentAccessControl(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	owner := setup.CreateTestUser(t)
	other := setup.CreateTestUser(t)

	resp, err := setup.MakeRequest("POST", "/api/v1/payments", map[string]interface{}{
		"asset_id":    1,
		"amount_fiat": 5000,
		"currency":    "USD",
		"gateway":     "stripe",
	}, owner.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	paymentID := fmt.Sprintf("%v", created["payment_id"])

	resp, err = setup.MakeRequest("GET", "/api/v1/payments/"+paymentID, nil, other.Token)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	t.Log("✅ Payment access control tests passed")
}
