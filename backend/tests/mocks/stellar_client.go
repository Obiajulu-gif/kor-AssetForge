package mocks

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/stellar/go/keypair"
	protocol "github.com/stellar/go/protocols/rpc"
	"github.com/stellar/go/txnbuild"
)

// MockAccount implements txnbuild.Account for unit testing offline
type MockAccount struct {
	AccountID string
	SeqNum    int64
}

func (m *MockAccount) GetAccountID() string {
	return m.AccountID
}

func (m *MockAccount) IncrementSequenceNumber() (int64, error) {
	m.SeqNum++
	return m.SeqNum, nil
}

func (m *MockAccount) GetSequenceNumber() (int64, error) {
	return m.SeqNum, nil
}

// MockSorobanRPC implements utils.SorobanRPC interface for offline testing
type MockSorobanRPC struct {
	mu                   sync.RWMutex
	Accounts             map[string]*MockAccount
	SimulateResponse     protocol.SimulateTransactionResponse
	SimulateError        error
	SendResponse         protocol.SendTransactionResponse
	SendError            error
	GetResponses         map[string]protocol.GetTransactionResponse
	GetError             error
	SimulateCallCount    int
	SendCallCount        int
	GetCallCount         int
}

func NewMockSorobanRPC() *MockSorobanRPC {
	return &MockSorobanRPC{
		Accounts:     make(map[string]*MockAccount),
		GetResponses: make(map[string]protocol.GetTransactionResponse),
	}
}

func (m *MockSorobanRPC) LoadAccount(ctx context.Context, address string) (txnbuild.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if acc, ok := m.Accounts[address]; ok {
		return acc, nil
	}
	// Return a default mock account if not explicitly set
	return &MockAccount{
		AccountID: address,
		SeqNum:    100,
	}, nil
}

func (m *MockSorobanRPC) SimulateTransaction(ctx context.Context, req protocol.SimulateTransactionRequest) (protocol.SimulateTransactionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SimulateCallCount++

	if m.SimulateError != nil {
		return protocol.SimulateTransactionResponse{}, m.SimulateError
	}
	return m.SimulateResponse, nil
}

func (m *MockSorobanRPC) SendTransaction(ctx context.Context, req protocol.SendTransactionRequest) (protocol.SendTransactionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SendCallCount++

	if m.SendError != nil {
		return protocol.SendTransactionResponse{}, m.SendError
	}
	if m.SendResponse.Hash != "" {
		return m.SendResponse, nil
	}
	return protocol.SendTransactionResponse{
		Status: "PENDING",
		Hash:   "mock-tx-hash-1234567890abcdef",
	}, nil
}

func (m *MockSorobanRPC) GetTransaction(ctx context.Context, req protocol.GetTransactionRequest) (protocol.GetTransactionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetCallCount++

	if m.GetError != nil {
		return protocol.GetTransactionResponse{}, m.GetError
	}
	if resp, ok := m.GetResponses[req.Hash]; ok {
		return resp, nil
	}
	return protocol.GetTransactionResponse{
		Status: protocol.TransactionStatusSuccess,
	}, nil
}

// MockStellarClient simulates high-level Stellar operations without connecting to Horizon
type MockStellarClient struct {
	mu              sync.RWMutex
	Balances        map[string]string
	ValidAddresses  map[string]bool
	ContractResults map[string]string
	ContractErrors  map[string]error
	SimulateOffline bool
	Calls           []string
}

func NewMockStellarClient() *MockStellarClient {
	return &MockStellarClient{
		Balances:        make(map[string]string),
		ValidAddresses:  make(map[string]bool),
		ContractResults: make(map[string]string),
		ContractErrors:  make(map[string]error),
	}
}

func (m *MockStellarClient) SetBalance(address, balance string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Balances[address] = balance
}

func (m *MockStellarClient) ValidateAddress(address string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.SimulateOffline {
		return errors.New("network unreachable: horizon offline")
	}

	if valid, ok := m.ValidAddresses[address]; ok {
		if !valid {
			return errors.New("invalid stellar address")
		}
		return nil
	}

	// Default validation via keypair parser
	_, err := keypair.ParseAddress(address)
	return err
}

func (m *MockStellarClient) GetAccountBalance(address string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, fmt.Sprintf("GetAccountBalance:%s", address))

	if m.SimulateOffline {
		return "", errors.New("horizon service unavailable (503)")
	}

	if bal, ok := m.Balances[address]; ok {
		return bal, nil
	}
	return "100.0000000", nil
}

func (m *MockStellarClient) InvokeContract(contractID string, method string, params []interface{}) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", contractID, method)
	m.Calls = append(m.Calls, fmt.Sprintf("InvokeContract:%s", key))

	if m.SimulateOffline {
		return "", errors.New("soroban RPC endpoint unreachable")
	}

	if err, ok := m.ContractErrors[key]; ok && err != nil {
		return "", err
	}

	if hash, ok := m.ContractResults[key]; ok {
		return hash, nil
	}

	return fmt.Sprintf("mock-tx-settled-%s-%s", contractID[:min(8, len(contractID))], method), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
