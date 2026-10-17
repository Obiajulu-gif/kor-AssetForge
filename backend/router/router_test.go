package router

import (
	"testing"

	"github.com/stellar/go/strkey"
	"github.com/stretchr/testify/require"
)

func TestSetupRouterRejectsInvalidConfigBeforeServices(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("SOROBAN_ASSET_TOKEN_CONTRACT_ID", strkey.MustEncode(strkey.VersionByteContract, make([]byte, 32)))
	require.Panics(t, func() { SetupRouter(nil) })
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("SOROBAN_ASSET_TOKEN_CONTRACT_ID", "invalid")
	require.Panics(t, func() { SetupRouter(nil) })
}
