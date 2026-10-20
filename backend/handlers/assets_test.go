package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stellar/go/strkey"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/config"
	"github.com/yourusername/kor-assetforge/models"
	"github.com/yourusername/kor-assetforge/validator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type recordingContractInvoker struct {
	contractID, method string
}

func (s *recordingContractInvoker) InvokeContract(contractID, method string, _ []interface{}) (string, error) {
	s.contractID, s.method = contractID, method
	return "test-transaction-hash", nil
}

func TestNewAssetHandlerRejectsInvalidContractID(t *testing.T) {
	for _, id := range []string{"", "CXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", "invalid"} {
		handler, err := NewAssetHandler(nil, nil, nil, nil, id)
		require.ErrorContains(t, err, "SOROBAN_ASSET_TOKEN_CONTRACT_ID")
		require.Nil(t, handler)
	}
}

func TestTokenizeAssetUsesConfiguredContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, validator.Init())
	contractID := strkey.MustEncode(strkey.VersionByteContract, bytes.Repeat([]byte{42}, 32))
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("SOROBAN_ASSET_TOKEN_CONTRACT_ID", contractID)
	cfg, err := config.LoadRuntimeConfig()
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&models.Asset{}, &models.FractionalizationConfig{}))
	invoker := &recordingContractInvoker{}
	handler, err := NewAssetHandler(db, invoker, nil, nil, cfg.AssetTokenContractID)
	require.NoError(t, err)
	router := gin.New()
	router.POST("/assets", handler.TokenizeAsset)
	payload, err := json.Marshal(validator.TokenizeAssetRequest{
		IssuerAccount: strkey.MustEncode(strkey.VersionByteAccountID, make([]byte, 32)),
		Name:          "Test Asset", Symbol: "TEST", AssetType: "real_estate", TotalSupply: 1000,
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/assets", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.Equal(t, contractID, invoker.contractID)
	require.Equal(t, "mint", invoker.method)
}

func TestTokenizeAssetRequestValidation(t *testing.T) {
	if err := validator.Init(); err != nil {
		t.Fatalf("failed to initialize validator: %v", err)
	}

	req := validator.TokenizeAssetRequest{
		IssuerAccount: "GABXYMNLGGTWAV7EQYHVWLQJ7MSAKBW3OX4J5B3UALQTVOUGX3HXBMEM",
		Name:          "Real Asset",
		Symbol:        "RWA1",
		AssetType:     "real_estate",
		TotalSupply:   1000,
	}

	if err := validator.ValidateStruct(&req); err != nil {
		t.Fatalf("expected valid request to pass validation: %v", err)
	}
}

func TestTokenizeAssetRequestRejectsInvalidIssuer(t *testing.T) {
	if err := validator.Init(); err != nil {
		t.Fatalf("failed to initialize validator: %v", err)
	}

	req := validator.TokenizeAssetRequest{
		IssuerAccount: "INVALIDADDRESS",
		Name:          "Real Asset",
		Symbol:        "RWA1",
		AssetType:     "real_estate",
		TotalSupply:   1000,
	}

	if err := validator.ValidateStruct(&req); err == nil {
		t.Fatal("expected invalid issuer address to fail validation")
	}
}
