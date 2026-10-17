package middleware

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCSRFProtectionTokenValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "0123456789abcdef0123456789abcdef"
	cases := []struct {
		name, value string
		status      int
	}{
		{"valid", secret, http.StatusOK},
		{"missing", "", http.StatusForbidden},
		{"first byte differs", "x" + secret[1:], http.StatusForbidden},
		{"middle byte differs", secret[:16] + "x" + secret[17:], http.StatusForbidden},
		{"last byte differs", secret[:31] + "x", http.StatusForbidden},
		{"prefix only", secret[:31], http.StatusForbidden},
		{"longer", secret + "x", http.StatusForbidden},
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, tc := range cases {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				router := gin.New()
				router.Use(CSRFProtection(secret))
				router.Handle(method, "/test", func(c *gin.Context) { c.Status(http.StatusOK) })
				req := httptest.NewRequest(method, "/test", nil)
				req.Header.Set("X-CSRF-Token", tc.value)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, req)
				if response.Code != tc.status {
					t.Fatalf("expected %d, got %d", tc.status, response.Code)
				}
			})
		}
	}
	for _, tc := range []struct{ method, secret string }{
		{http.MethodGet, secret}, {http.MethodHead, secret},
		{http.MethodOptions, secret}, {http.MethodPost, ""},
	} {
		router := gin.New()
		router.Use(CSRFProtection(tc.secret))
		router.Handle(tc.method, "/test", func(c *gin.Context) { c.Status(http.StatusOK) })
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(tc.method, "/test", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("unexpected rejection for %s", tc.method)
		}
	}
}

// Structural regression check: wall-clock timing assertions are noisy and cannot
// prove constant-time behavior. Require the audited standard-library primitive
// and reject a reintroduction of direct equality against the supplied token.
func TestCSRFProtectionUsesConstantTimeComparison(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "security.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "CSRFProtection" {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					pkg, ok := selector.X.(*ast.Ident)
					if ok && pkg.Name == "subtle" && selector.Sel.Name == "ConstantTimeCompare" {
						found = true
					}
				}
			}
			if expr, ok := node.(*ast.BinaryExpr); ok && (expr.Op == token.EQL || expr.Op == token.NEQ) {
				for _, operand := range []ast.Expr{expr.X, expr.Y} {
					if id, ok := operand.(*ast.Ident); ok && id.Name == "token" {
						t.Error("CSRF token must not use direct string equality")
					}
				}
			}
			return true
		})
	}
	if !found {
		t.Error("CSRFProtection must use subtle.ConstantTimeCompare")
	}
}

func TestRequireJSONRejectsUnsupportedMediaType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireJSON())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("hello"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected %d status, got %d", http.StatusUnsupportedMediaType, w.Code)
	}
}

func TestCSRFProtectionRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection("secret-token"))
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d status, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRateLimitBlocksExcessRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimit(2, time.Second))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected OK on request %d, got %d", i+1, w.Code)
		}
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d status on rate limit exceed, got %d", http.StatusTooManyRequests, w.Code)
	}
}
