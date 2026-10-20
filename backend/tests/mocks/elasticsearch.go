package mocks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/yourusername/kor-assetforge/models"
)

// MockESTransport intercepts HTTP calls to Elasticsearch and provides in-memory mock responses.
type MockESTransport struct {
	mu             sync.RWMutex
	IndexedDocs    map[string][]byte
	ShouldFail     bool
	FailStatusCode int
	FailError      error
	RequestCount   int
}

func NewMockESTransport() *MockESTransport {
	return &MockESTransport{
		IndexedDocs: make(map[string][]byte),
	}
}

func (t *MockESTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.RequestCount++

	if t.ShouldFail {
		if t.FailError != nil {
			return nil, t.FailError
		}
		status := t.FailStatusCode
		if status == 0 {
			status = http.StatusServiceUnavailable
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error": "elasticsearch offline or unavailable"}`)),
			Header:     make(http.Header),
		}, nil
	}

	// Read body if bulk index or index request
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		t.IndexedDocs[req.URL.Path] = bodyBytes
	}

	// Successful mock response
	respBody := `{"acknowledged": true, "items": [], "took": 1, "errors": false}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(respBody)),
		Header:     make(http.Header),
	}, nil
}

// NewMockElasticsearchClient creates an elasticsearch.Client powered by an in-memory MockESTransport.
func NewMockElasticsearchClient(transport *MockESTransport) (*elasticsearch.Client, error) {
	if transport == nil {
		transport = NewMockESTransport()
	}
	return elasticsearch.NewClient(elasticsearch.Config{
		Transport: transport,
		Addresses: []string{"http://mock-elasticsearch:9200"},
	})
}

// MockElasticsearchIndexer simulates indexing operations without Elasticsearch.
type MockElasticsearchIndexer struct {
	mu           sync.RWMutex
	Indexed      []*models.Asset
	ShouldFail   bool
	ErrorMessage string
	Reindexed    bool
}

func NewMockElasticsearchIndexer() *MockElasticsearchIndexer {
	return &MockElasticsearchIndexer{
		Indexed: make([]*models.Asset, 0),
	}
}

func (m *MockElasticsearchIndexer) IndexAssetsBulk(ctx context.Context, assets []*models.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		msg := m.ErrorMessage
		if msg == "" {
			msg = "elasticsearch cluster unavailable: connection refused"
		}
		return errors.New(msg)
	}

	m.Indexed = append(m.Indexed, assets...)
	return nil
}

func (m *MockElasticsearchIndexer) ReindexAllAssets(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		return fmt.Errorf("reindex failed: %s", m.ErrorMessage)
	}

	m.Reindexed = true
	return nil
}
