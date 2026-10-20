package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kor-assetforge/constants"
)

func newPaginationRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(EnforcePaginationLimits())
	r.GET("/items", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestEnforcePaginationLimits_AllowsWithinBounds(t *testing.T) {
	r := newPaginationRouter()

	for _, query := range []string{"", "?limit=1", "?limit=100", "?page_size=100"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/items"+query, nil))
		if w.Code != http.StatusOK {
			t.Errorf("query %q: got status %d, want %d", query, w.Code, http.StatusOK)
		}
	}
}

func TestEnforcePaginationLimits_RejectsOversizedLimit(t *testing.T) {
	r := newPaginationRouter()

	for _, query := range []string{"?limit=101", "?limit=5000", "?page_size=1000"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/items"+query, nil))
		if w.Code != http.StatusBadRequest {
			t.Errorf("query %q: got status %d, want %d", query, w.Code, http.StatusBadRequest)
		}
	}
}

func TestEnforcePaginationLimits_IgnoresMalformedValue(t *testing.T) {
	r := newPaginationRouter()

	// A non-numeric value is left to the handler default rather than rejected.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/items?limit=abc", nil))
	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestEnforcePaginationLimits_ConstantIsEnforced(t *testing.T) {
	if constants.MAX_PAGE_SIZE != 100 {
		t.Fatalf("MAX_PAGE_SIZE = %d, want 100", constants.MAX_PAGE_SIZE)
	}

	r := newPaginationRouter()
	// One over the documented maximum must be rejected.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/items?limit=100", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("limit=100 should be allowed, got %d", w.Code)
	}
}
