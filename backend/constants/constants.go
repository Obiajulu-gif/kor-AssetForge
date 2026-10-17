package constants

// Global operational constants for kor-AssetForge

const (
	// MAX_BATCH_SIZE is the global maximum number of operations allowed in a single batch transaction.
	MAX_BATCH_SIZE = 50

	// DefaultMaxBatchSize provides a default fallback for batch size limits.
	DefaultMaxBatchSize = 50

	// MAX_PAGE_SIZE is the global maximum number of records a client may request
	// from any paginated list endpoint. Requests that exceed this bound are
	// rejected with HTTP 400 by the pagination middleware.
	MAX_PAGE_SIZE = 100

	// DefaultPageSize is the page size applied when a client does not supply one.
	DefaultPageSize = 10
)
