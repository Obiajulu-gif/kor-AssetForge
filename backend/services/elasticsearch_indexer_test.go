package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/models"
	"github.com/yourusername/kor-assetforge/tests/mocks"
)

func TestElasticsearchIndexer_IndexAssetsBulk_Offline(t *testing.T) {
	transport := mocks.NewMockESTransport()
	client, err := mocks.NewMockElasticsearchClient(transport)
	require.NoError(t, err)

	indexer := NewElasticsearchIndexer(client, nil)

	assets := []*models.Asset{
		{
			ID:          1,
			Name:        "Test Commercial Tower",
			Symbol:      "TCT01",
			Description: "High yield commercial asset",
			AssetType:   "real_estate",
			TotalSupply: 1000000,
			Fractions:   7,
			ContractID:  "CA11111111111111111111111111111111111111111111111111111111",
			Verified:    true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          2,
			Name:        "Green Solar Farm",
			Symbol:      "GSF02",
			Description: "Renewable energy infrastructure",
			AssetType:   "energy",
			TotalSupply: 500000,
			Fractions:   7,
			ContractID:  "CA22222222222222222222222222222222222222222222222222222222",
			Verified:    true,
			CreatedAt:   time.Now(),
		},
	}

	ctx := context.Background()
	err = indexer.IndexAssetsBulk(ctx, assets)
	assert.NoError(t, err, "Offline Elasticsearch bulk indexing should succeed with mock transport")
	assert.GreaterOrEqual(t, transport.RequestCount, 1, "Mock transport should record bulk indexing requests")
}

func TestElasticsearchIndexer_ErrorScenario(t *testing.T) {
	transport := mocks.NewMockESTransport()
	transport.ShouldFail = true
	transport.FailStatusCode = 503
	transport.FailError = errors.New("connection refused to cluster")

	client, err := mocks.NewMockElasticsearchClient(transport)
	require.NoError(t, err)

	indexer := NewElasticsearchIndexer(client, nil)

	assets := []*models.Asset{
		{
			ID:   10,
			Name: "Faulty Asset",
		},
	}

	ctx := context.Background()
	err = indexer.IndexAssetsBulk(ctx, assets)
	assert.Error(t, err, "Bulk indexing must return an error when Elasticsearch is unavailable")
	assert.Contains(t, err.Error(), "bulk indexing failed")
}

func TestElasticsearchIndexer_NilClientHandling(t *testing.T) {
	indexer := NewElasticsearchIndexer(nil, nil)
	ctx := context.Background()

	err := indexer.IndexAssetsBulk(ctx, []*models.Asset{{ID: 1}})
	assert.NoError(t, err, "Nil client should return nil gracefully without crashing")

	err = indexer.ReindexAllAssets(ctx)
	assert.NoError(t, err, "Nil client reindexing should return nil gracefully")
}
