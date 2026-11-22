package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sublink/middlewares"
	"sublink/models"
	"sublink/security"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginAuditRequiresAuthenticationAndRecordsActualOutcome(t *testing.T) {
	testDatabase(t)
	ctx, cancel := context.WithCancel(context.Background())
	previous := LoginAudit
	t.Cleanup(func() { cancel(); LoginAudit = previous })
	var err error
	LoginAudit, err = security.New(ctx, filepath.Join(t.TempDir(), "alerts.json"), func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "admin", Password: "not-for-audit-records"}
	if err := user.Create(); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(middlewares.AuthorToken)
	r.POST("/api/v1/auth/login", UserLogin)
	r.GET("/api/v1/users/login-history", LoginHistory)
	r.GET("/api/v1/users/login-notifications", LoginNotificationConfig)
	r.PUT("/api/v1/users/login-notifications", SaveLoginNotificationConfig)
	r.POST("/api/v1/users/login-notifications/test", TestLoginNotification)
	for _, route := range []struct{ method, path string }{
		{"GET", "/api/v1/users/login-history"}, {"GET", "/api/v1/users/login-notifications"},
		{"PUT", "/api/v1/users/login-notifications"}, {"POST", "/api/v1/users/login-notifications/test"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(route.method, route.path, nil))
		if w.Code < 400 {
			t.Fatal("unauthenticated audit access", route.path)
		}
	}
	var token string
	for _, password := range []string{"wrong-password", "not-for-audit-records"} {
		form := url.Values{"username": {"admin"}, "password": {password}}
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(form.Encode()))
		req.RemoteAddr = "9.9.9.9:12345"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("CF-Connecting-IP", "8.8.8.8")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if password == "wrong-password" {
			if w.Code != 400 {
				t.Fatal("wrong password accepted")
			}
		} else {
			var body struct {
				Data struct {
					AccessToken string `json:"accessToken"`
				} `json:"data"`
			}
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
				t.Fatal("login failed", w.Body.String())
			}
			token = body.Data.AccessToken
		}
	}
	// LAN logins still authenticate normally; neither outcome enters public audit.
	for _, password := range []string{"wrong-password", "not-for-audit-records"} {
		form := url.Values{"username": {"admin"}, "password": {password}}
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(form.Encode()))
		req.RemoteAddr = "192.168.1.10:12345"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("CF-Connecting-IP", "8.8.8.8")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		want := 200
		if password == "wrong-password" {
			want = 400
		}
		if w.Code != want || len(LoginAudit.Snapshot().List) != 2 {
			t.Fatal("LAN authentication changed or created a public login record")
		}
	}
	req := httptest.NewRequest("GET", "/api/v1/users/login-history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var response struct {
		Data security.Snapshot `json:"data"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &response) != nil {
		t.Fatal("audit query failed")
	}
	if len(response.Data.List) != 2 || !response.Data.List[0].Success || response.Data.List[1].Success || response.Data.List[0].IP != "9.9.9.9" {
		t.Fatal("wrong login outcome/IP", w.Body.String())
	}
	for _, secret := range []string{token, "wrong-password", "not-for-audit-records"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("credentials in audit history")
		}
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("audit data can be cached")
	}
}
