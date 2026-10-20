package handlers

import "github.com/yourusername/kor-assetforge/constants"

// clampPageLimit bounds a caller-supplied page size to the global maximum
// defined in constants.MAX_PAGE_SIZE. It is a defence-in-depth guard used by
// list handlers in addition to the global pagination middleware (#302):
// non-positive values fall back to constants.DefaultPageSize and oversized
// values are capped so a handler can never issue an unbounded query.
func clampPageLimit(limit int) int {
	if limit <= 0 {
		return constants.DefaultPageSize
	}
	if limit > constants.MAX_PAGE_SIZE {
		return constants.MAX_PAGE_SIZE
	}
	return limit
}
