package config

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/stellar/go/strkey"
	"github.com/stretchr/testify/require"
)

func TestValidateJWTSecret(t *testing.T) {
	for _, tc := range []struct {
		name, secret, message string
	}{
		{"missing", "", "JWT_SECRET is required"},
		{"whitespace", strings.Repeat(" ", 40), "JWT_SECRET is required"},
		{"short", strings.Repeat("a", 31), "at least 32 characters"},
		{"short unicode", strings.Repeat("é", 31), "at least 32 characters"},
		{"minimum", "0123456789abcdef0123456789abcdef", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJWTSecret(tc.secret)
			if tc.message == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.message)
			}
		})
	}
}

func TestValidateAssetTokenContractID(t *testing.T) {
	valid := strkey.MustEncode(strkey.VersionByteContract, bytes.Repeat([]byte{42}, 32))
	for _, tc := range []struct {
		name, id string
		valid    bool
	}{
		{"valid contract", valid, true},
		{"missing", "", false},
		{"placeholder", "C" + strings.Repeat("X", 55), false},
		{"account address", strkey.MustEncode(strkey.VersionByteAccountID, make([]byte, 32)), false},
		{"bad checksum", valid[:55] + "A", false},
		{"lowercase", strings.ToLower(valid), false},
		{"space", " " + valid, false},
		{"short payload with valid checksum", strkey.MustEncode(strkey.VersionByteContract, make([]byte, 31)), false},
		{"long payload with valid checksum", strkey.MustEncode(strkey.VersionByteContract, make([]byte, 33)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAssetTokenContractID(tc.id)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "SOROBAN_ASSET_TOKEN_CONTRACT_ID")
			}
		})
	}
}

func TestLoadRuntimeConfig(t *testing.T) {
	validID := strkey.MustEncode(strkey.VersionByteContract, make([]byte, 32))
	t.Setenv("SOROBAN_ASSET_TOKEN_CONTRACT_ID", validID)
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	cfg, err := LoadRuntimeConfig()
	require.NoError(t, err)
	require.Equal(t, validID, cfg.AssetTokenContractID)
	require.Equal(t, "0123456789abcdef0123456789abcdef", cfg.JWTSecret)

	t.Setenv("JWT_SECRET", "")
	_, err = LoadRuntimeConfig()
	require.ErrorContains(t, err, "JWT_SECRET is required")
	t.Setenv("JWT_SECRET", "short")
	_, err = LoadRuntimeConfig()
	require.ErrorContains(t, err, "at least 32 characters")
	t.Setenv("JWT_SECRET", cfg.JWTSecret)
	for _, id := range []string{"", "invalid"} {
		t.Setenv("SOROBAN_ASSET_TOKEN_CONTRACT_ID", id)
		_, err = LoadRuntimeConfig()
		require.ErrorContains(t, err, "SOROBAN_ASSET_TOKEN_CONTRACT_ID")
	}
}

func TestLoadRuntimeConfigWarnsWithoutLoggingSecret(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	t.Setenv("SOROBAN_ASSET_TOKEN_CONTRACT_ID", strkey.MustEncode(strkey.VersionByteContract, make([]byte, 32)))
	for _, tc := range []struct {
		secret string
		weak   bool
	}{
		{"your-super-secret-jwt-key-change-in-production", true},
		{strings.Repeat("a", 32), true},
		{"0123456789abcdef0123456789abcdef", false},
	} {
		output.Reset()
		t.Setenv("JWT_SECRET", tc.secret)
		_, err := LoadRuntimeConfig()
		require.NoError(t, err)
		require.Equal(t, tc.weak, strings.Contains(output.String(), "JWT_SECRET appears weak"))
		require.NotContains(t, output.String(), tc.secret)
	}
}
