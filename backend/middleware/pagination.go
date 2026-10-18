package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kor-assetforge/constants"
)

// pageSizeParams are the query parameters that select the page size on list
// endpoints. Both spellings are accepted so the limit is enforced no matter
// which convention a client (or handler) uses.
var pageSizeParams = []string{"limit", "page_size"}

// EnforcePaginationLimits is a reusable middleware that rejects list requests
// whose requested page size exceeds constants.MAX_PAGE_SIZE with HTTP 400.
//
// A missing parameter is allowed and left to the handler's default. A value
// that cannot be parsed as an integer is also allowed through so the existing
// handler-level defaults keep working; only an explicit oversized value is
// rejected. This guarantees a hard upper bound on result-set size while
// remaining backwards compatible with endpoints that ignore the parameter.
func EnforcePaginationLimits() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, param := range pageSizeParams {
			raw := c.Query(param)
			if raw == "" {
				continue
			}

			value, err := strconv.Atoi(raw)
			if err != nil {
				// Defer to the handler's default for malformed input.
				continue
			}

			if value > constants.MAX_PAGE_SIZE {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":   "requested page size exceeds the maximum allowed",
					"param":   param,
					"max":     constants.MAX_PAGE_SIZE,
					"message": "reduce limit/page_size to " + strconv.Itoa(constants.MAX_PAGE_SIZE) + " or fewer",
				})
				return
			}
		}
		c.Next()
	}
}
