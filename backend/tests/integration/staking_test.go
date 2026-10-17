//go:build ignore
// +build ignore

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStakingFlow tests staking, dashboard and unstaking.
func TestStakingFlow(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)
	setup.ApproveKYC(t, user.PublicKey)

	// Step 1: stake tokens against an asset
	stakeData := map[string]interface{}{
		"user_id":         user.UserID,
		"asset_id":        1,
		"stellar_address": user.PublicKey,
		"amount":          1000000,
	}

	resp, err := setup.MakeRequest("POST", "/api/v1/staking/stake", stakeData, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var position map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&position))
	assert.Equal(t, float64(1000000), position["staked_amount"])
	assert.Equal(t, true, position["active"])

	// Step 2: staking dashboard reflects the position
	resp, err = setup.MakeRequest("GET", "/api/v1/staking/dashboard", nil, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var dashboard map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&dashboard))
	assert.Contains(t, dashboard, "total_staked")

	// Step 3: unstake part of the position
	resp, err = setup.MakeRequest("POST", "/api/v1/staking/unstake", map[string]interface{}{
		"user_id":  user.UserID,
		"asset_id": 1,
		"amount":   500000,
	}, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	t.Log("✅ Staking flow completed successfully")
}

// TestStakingTopUp tests that staking again tops up an existing position.
func TestStakingTopUp(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)

	first := map[string]interface{}{
		"user_id":         user.UserID,
		"asset_id":        1,
		"stellar_address": user.PublicKey,
		"amount":          1000000,
	}
	resp, err := setup.MakeRequest("POST", "/api/v1/staking/stake", first, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	second := map[string]interface{}{
		"user_id":         user.UserID,
		"asset_id":        1,
		"stellar_address": user.PublicKey,
		"amount":          250000,
	}
	resp, err = setup.MakeRequest("POST", "/api/v1/staking/stake", second, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var position map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&position))
	assert.Equal(t, float64(1250000), position["staked_amount"])

	t.Log("✅ Staking top-up tests passed")
}

// TestStakingRewards tests claiming rewards and reading reward history.
func TestStakingRewards(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)
	setup.ApproveKYC(t, user.PublicKey)

	resp, err := setup.MakeRequest("POST", "/api/v1/staking/stake", map[string]interface{}{
		"user_id":         user.UserID,
		"asset_id":        1,
		"stellar_address": user.PublicKey,
		"amount":          1000000,
	}, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Claim rewards for the position
	resp, err = setup.MakeRequest("POST", "/api/v1/staking/claim", map[string]interface{}{
		"user_id":  user.UserID,
		"asset_id": 1,
	}, user.Token)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
		"unexpected claim status %d", resp.StatusCode)

	// Reward history endpoint
	resp, err = setup.MakeRequest("GET", "/api/v1/staking/rewards/history", nil, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	t.Log("✅ Staking reward tests passed")
}

// TestStakingValidations tests staking input validation.
func TestStakingValidations(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)

	testCases := []struct {
		name   string
		data   map[string]interface{}
		status int
	}{
		{
			name: "missing amount",
			data: map[string]interface{}{
				"user_id":         user.UserID,
				"asset_id":        1,
				"stellar_address": user.PublicKey,
			},
			status: http.StatusBadRequest,
		},
		{
			name: "zero amount",
			data: map[string]interface{}{
				"user_id":         user.UserID,
				"asset_id":        1,
				"stellar_address": user.PublicKey,
				"amount":          0,
			},
			status: http.StatusBadRequest,
		},
		{
			name: "unknown asset",
			data: map[string]interface{}{
				"user_id":         user.UserID,
				"asset_id":        999999,
				"stellar_address": user.PublicKey,
				"amount":          1000,
			},
			status: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := setup.MakeRequest("POST", "/api/v1/staking/stake", tc.data, user.Token)
			require.NoError(t, err)
			assert.Equal(t, tc.status, resp.StatusCode)
		})
	}

	t.Log("✅ Staking validation tests passed")
}

// TestStakingUnstakeMoreThanStaked tests that over-unstaking is rejected.
func TestStakingUnstakeMoreThanStaked(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	user := setup.CreateTestUser(t)

	resp, err := setup.MakeRequest("POST", "/api/v1/staking/stake", map[string]interface{}{
		"user_id":         user.UserID,
		"asset_id":        1,
		"stellar_address": user.PublicKey,
		"amount":          1000000,
	}, user.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err = setup.MakeRequest("POST", "/api/v1/staking/unstake", map[string]interface{}{
		"user_id":  user.UserID,
		"asset_id": 1,
		"amount":   9999999,
	}, user.Token)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	t.Log("✅ Over-unstake rejection test passed")
}
