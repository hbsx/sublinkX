package middlewares

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthenticationWhitelistUsesExactRoutesAndMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, route := range []struct {
		method, path string
		public       bool
	}{
		{"POST", "/api/v1/auth/login", true}, {"GET", "/api/v1/version", true}, {"GET", "/c/", true},
		{"GET", "/static/js/app.js", true}, {"GET", "/", true},
		{"GET", "/api/v1/auth/login-history", false}, {"GET", "/api/v1/auth/captcha-private", false},
		{"GET", "/api/v1/version-private", false}, {"GET", "/static-private", false},
		{"PUT", "/api/v1/version", false}, {"DELETE", "/c/", false},
	} {
		r := gin.New()
		r.Use(AuthorToken)
		r.Handle(route.method, route.path, func(c *gin.Context) { c.Status(204) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(route.method, route.path, nil))
		if (w.Code == 204) != route.public {
			t.Errorf("%s %s: unexpected status %d", route.method, route.path, w.Code)
		}
	}
}

func TestOversizedAndDotFloodTokensAreRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthorToken)
	r.GET("/private", func(c *gin.Context) { t.Error("invalid token reached handler") })
	for _, token := range []string{strings.Repeat(".", 100000), strings.Repeat("x", 9000) + ".y.z", "a.b.c.d"} {
		request := httptest.NewRequest("GET", "/private", nil)
		request.Header.Set("Authorization", token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)
		if w.Code != 400 {
			t.Errorf("invalid token returned %d", w.Code)
		}
		if _, err := ParseToken(token); err == nil {
			t.Fatal("parser accepted invalid token")
		}
	}
}
