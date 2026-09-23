package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestAuthMiddleware exercises the token gate directly: an empty token must
// leave routes open (localhost dev), and a set token must reject both a missing
// and a wrong bearer token while accepting the right one.
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		token      string
		authHeader string
		wantStatus int
	}{
		{"no token configured allows through", "", "", http.StatusOK},
		{"no token configured ignores header", "", "Bearer whatever", http.StatusOK},
		{"set token rejects missing header", "s3cret", "", http.StatusUnauthorized},
		{"set token rejects wrong token", "s3cret", "Bearer wrong", http.StatusUnauthorized},
		{"set token rejects missing Bearer prefix", "s3cret", "s3cret", http.StatusUnauthorized},
		{"set token accepts correct token", "s3cret", "Bearer s3cret", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(authMiddleware(tc.token))
			r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}
