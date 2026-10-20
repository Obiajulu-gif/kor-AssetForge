package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	protocol "github.com/stellar/go/protocols/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/tests/mocks"
)

// TestMockStellarClient_OfflineExecution verifies that Stellar operations execute offline without Horizon.
func TestMockStellarClient_OfflineExecution(t *testing.T) {
	mockStellar := mocks.NewMockStellarClient()

	pair, err := keypair.Random()
	require.NoError(t, err)
	address := pair.Address()

	// 1. Validate Address offline
	err = mockStellar.ValidateAddress(address)
	assert.NoError(t, err, "Valid address should pass offline validation")

	// Invalid address format
	err = mockStellar.ValidateAddress("INVALID_STELLAR_ADDRESS")
	assert.Error(t, err, "Invalid address must fail validation")

	// 2. Query balance offline
	mockStellar.SetBalance(address, "2500.5000000")
	bal, err := mockStellar.GetAccountBalance(address)
	require.NoError(t, err)
	assert.Equal(t, "2500.5000000", bal)

	// 3. Invoke contract offline
	txHash, err := mockStellar.InvokeContract("CA3D5KRYMCMV244BH7AVWNDCZWYCXNOKZRI2UMQIO2GTJ36K6IZ5MCPO", "transfer", []interface{}{address, int64(100)})
	require.NoError(t, err)
	assert.NotEmpty(t, txHash)
}

// TestMockStellarClient_ErrorScenarios verifies simulated errors when Stellar services are degraded.
func TestMockStellarClient_ErrorScenarios(t *testing.T) {
	mockStellar := mocks.NewMockStellarClient()
	mockStellar.SimulateOffline = true

	_, err := mockStellar.GetAccountBalance("GBTEST")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "horizon service unavailable")

	_, err = mockStellar.InvokeContract("CA1234", "mint", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "soroban RPC endpoint unreachable")
}

// TestMockSorobanRPC_SimulationAndSend verifies Soroban transaction RPC mocking.
func TestMockSorobanRPC_SimulationAndSend(t *testing.T) {
	rpc := mocks.NewMockSorobanRPC()
	ctx := context.Background()

	// Simulate contract call
	rpc.SimulateResponse = protocol.SimulateTransactionResponse{
		MinResourceFee: 1000,
	}
	sim, err := rpc.SimulateTransaction(ctx, protocol.SimulateTransactionRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(1000), sim.MinResourceFee)

	// Simulated failure
	rpc.SimulateError = errors.New("contract execution reverted: insufficient allowance")
	_, err = rpc.SimulateTransaction(ctx, protocol.SimulateTransactionRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient allowance")
}

// TestMockEmailService_OfflineOperations verifies email dispatch without connecting to external providers.
func TestMockEmailService_OfflineOperations(t *testing.T) {
	emailMock := mocks.NewMockEmailService()

	// 1. Send verification email
	err := emailMock.SendVerificationEmail("trader@example.com", "Trader", "token-xyz-123")
	require.NoError(t, err)
	assert.Equal(t, 1, emailMock.TotalSent())

	// 2. Send KYC status update
	err = emailMock.SendKYCStatusUpdate("investor@example.com", "Investor", "approved", "Documents verified successfully")
	require.NoError(t, err)
	assert.Equal(t, 2, emailMock.TotalSent())

	// 3. Send transaction confirmation
	err = emailMock.SendTransactionConfirmation("buyer@example.com", "Buyer", "0xabc123", 50000, 1, "GADDR1", "GADDR2")
	require.NoError(t, err)
	assert.Equal(t, 3, emailMock.TotalSent())

	// Verify recorded email attributes
	last := emailMock.SentEmails[2]
	assert.Equal(t, "transaction_confirmation", last.Type)
	assert.Equal(t, "0xabc123", last.TxHash)
	assert.Equal(t, int64(50000), last.Amount)
}

// TestMockEmailService_ErrorScenarios tests transient failures and rate limits.
func TestMockEmailService_ErrorScenarios(t *testing.T) {
	emailMock := mocks.NewMockEmailService()

	// Simulate rate limit for 2 attempts
	emailMock.FailNextN = 2
	emailMock.FailError = errors.New("429 Too Many Requests")

	err := emailMock.SendVerificationEmail("test1@example.com", "Test1", "t1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "429 Too Many Requests")

	err = emailMock.SendVerificationEmail("test2@example.com", "Test2", "t2")
	assert.Error(t, err)

	// Third attempt should succeed automatically
	err = emailMock.SendVerificationEmail("test3@example.com", "Test3", "t3")
	assert.NoError(t, err)
	assert.Equal(t, 1, emailMock.TotalSent())
}
