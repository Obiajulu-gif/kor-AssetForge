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

// TestUBODeclarationFlow tests declaring, listing and resolving a beneficial owner.
func TestUBODeclarationFlow(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	entity := setup.CreateTestUser(t)
	setup.ApproveKYC(t, entity.PublicKey)

	// Step 1: declare a beneficial owner
	declaration := map[string]interface{}{
		"entity_user_id":       entity.UserID,
		"full_name":            "Jane Holding",
		"date_of_birth":        "1978-04-12",
		"nationality":          "US",
		"ownership_percent":    60.0,
		"is_controlling_party": true,
		"ownership_layer":      1,
		"document_number":      "P99887766",
	}

	resp, err := setup.MakeRequest("POST", "/api/v1/compliance/ubo", declaration, entity.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var owner map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&owner))
	assert.Equal(t, "declared", owner["status"])
	assert.NotNil(t, owner["id"])

	// Step 2: list the entity's beneficial owners
	resp, err = setup.MakeRequest("GET", fmt.Sprintf("/api/v1/compliance/ubo?entity_user_id=%d", entity.UserID), nil, entity.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listing map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listing))
	assert.Contains(t, listing, "owners")
	assert.Contains(t, listing, "requires_ubo_program")

	t.Log("✅ UBO declaration flow completed successfully")
}

// TestUBOValidation tests ownership-percent validation across declarations.
func TestUBOValidation(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	entity := setup.CreateTestUser(t)

	// An implausible ownership percentage must be rejected.
	resp, err := setup.MakeRequest("POST", "/api/v1/compliance/ubo", map[string]interface{}{
		"entity_user_id":   entity.UserID,
		"full_name":        "Over Owner",
		"date_of_birth":    "1980-01-01",
		"nationality":      "US",
		"ownership_percent": 150.0,
		"ownership_layer":  1,
	}, entity.Token)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// A valid declaration succeeds.
	resp, err = setup.MakeRequest("POST", "/api/v1/compliance/ubo", map[string]interface{}{
		"entity_user_id":   entity.UserID,
		"full_name":        "Valid Owner",
		"date_of_birth":    "1980-01-01",
		"nationality":      "US",
		"ownership_percent": 40.0,
		"ownership_layer":  1,
	}, entity.Token)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	t.Log("✅ UBO validation tests passed")
}

// TestUBOReviewResolution tests the compliance-team resolve action.
func TestUBOReviewResolution(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	entity := setup.CreateTestUser(t)

	resp, err := setup.MakeRequest("POST", "/api/v1/compliance/ubo", map[string]interface{}{
		"entity_user_id":   entity.UserID,
		"full_name":        "Review Target",
		"date_of_birth":    "1975-09-09",
		"nationality":      "US",
		"ownership_percent": 30.0,
		"ownership_layer":  1,
	}, entity.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var owner map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&owner))
	ownerID := fmt.Sprintf("%d", uint(owner["id"].(float64)))

	// Admin verifies the declaration
	resp, err = setup.MakeRequest("POST", "/api/v1/compliance/ubo/"+ownerID+"/resolve", map[string]interface{}{
		"status":       "verified",
		"review_notes": "Documents verified",
	}, setup.AdminToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var resolved map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&resolved))
	assert.Equal(t, "verified", resolved["status"])

	t.Log("✅ UBO review resolution tests passed")
}

// TestCompliancePermissions ensures only admins can resolve UBO reviews.
func TestCompliancePermissions(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	entity := setup.CreateTestUser(t)

	resp, err := setup.MakeRequest("POST", "/api/v1/compliance/ubo", map[string]interface{}{
		"entity_user_id":   entity.UserID,
		"full_name":        "Permission Check",
		"date_of_birth":    "1982-02-02",
		"nationality":      "US",
		"ownership_percent": 25.0,
		"ownership_layer":  1,
	}, entity.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var owner map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&owner))
	ownerID := fmt.Sprintf("%d", uint(owner["id"].(float64)))

	// A regular user cannot resolve the review
	resp, err = setup.MakeRequest("POST", "/api/v1/compliance/ubo/"+ownerID+"/resolve", map[string]interface{}{
		"status": "verified",
	}, entity.Token)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	t.Log("✅ Compliance permission tests passed")
}
