package api

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sublink/middlewares"
	"sublink/models"
	"sublink/node"
	"sublink/settings"
)

func testDatabase(t *testing.T) {
	t.Helper()
	oldDirectory, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	oldDB, oldSecret := models.DB, middlewares.Secret
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	models.DB = db
	if err := db.AutoMigrate(&models.User{}, &models.Subcription{}, &models.Node{}, &models.GroupNode{}, &models.SubLogs{}); err != nil {
		t.Fatal(err)
	}
	if err := models.ConfigInit(); err != nil {
		t.Fatal(err)
	}
	config, _ := models.LoadConfig()
	middlewares.Secret = []byte(config.JwtSecret)
	gin.SetMode(gin.TestMode)
	sqlDB, _ := db.DB()
	t.Cleanup(func() {
		sqlDB.Close()
		models.DB, middlewares.Secret = oldDB, oldSecret
		os.Chdir(oldDirectory)
	})
}

func TestPasswordChangeAndResetRevokeTokensWithoutLoggingSecrets(t *testing.T) {
	testDatabase(t)
	admin := models.User{Username: "admin", Password: "old-password"}
	if err := admin.Create(); err != nil {
		t.Fatal(err)
	}
	token, err := GetToken("admin")
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(middlewares.AuthorToken)
	r.POST("/api/v1/users/update", UserSet)
	r.DELETE("/api/v1/auth/logout", UserOut)
	var output bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previousOutput)
	form := url.Values{"username": {"admin"}, "password": {"new-password-secret"}}
	request := httptest.NewRequest("POST", "/api/v1/users/update", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if strings.Contains(output.String(), "new-password-secret") {
		t.Fatal("password appeared in logs")
	}
	if _, err := middlewares.ParseToken(token); err == nil {
		t.Fatal("old login survived password change")
	}
	concurrentLogin, err := tokenForUser(&admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := middlewares.ParseToken(concurrentLogin); err == nil {
		t.Fatal("a login verified before the password change acquired a new session")
	}
	token, err = GetToken("admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := settings.ResetUser("admin", "replacement-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := middlewares.ParseToken(token); err == nil {
		t.Fatal("old login survived account reset")
	}
	token, _ = GetToken("admin")
	request = httptest.NewRequest("DELETE", "/api/v1/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if _, err := middlewares.ParseToken(token); err == nil {
		t.Fatal("old login survived logout")
	}
	login := models.User{Username: "admin", Password: "replacement-password"}
	if err := login.Verify(); err != nil {
		t.Fatal("replacement credentials failed", err)
	}
}

func TestMigrationKeepsDataAndTokensStable(t *testing.T) {
	testDatabase(t)
	password := strings.Repeat("long-password-", 10)
	legacy := models.User{Username: "legacy", Password: password}
	if err := models.DB.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	sub := models.Subcription{Name: "legacy-sub", Config: "{}"}
	if err := models.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	if err := models.MigrateSecurity(models.DB); err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "legacy", Password: password}
	if err := user.Verify(); err != nil {
		t.Fatal(err)
	}
	if user.Password == password {
		t.Fatal("legacy plaintext password was retained")
	}
	models.DB.First(&sub, sub.ID)
	token, version := sub.Token, user.SessionVersion
	if len(token) != 64 || version == "" {
		t.Fatal("migration did not create security credentials")
	}
	if err := models.MigrateSecurity(models.DB); err != nil {
		t.Fatal(err)
	}
	models.DB.First(&sub, sub.ID)
	user.Find()
	if sub.Token != token || user.SessionVersion != version {
		t.Fatal("restart rotated persisted credentials")
	}
}

func TestConcurrentSubscriptionsAreIsolatedAndMD5IsRejected(t *testing.T) {
	testDatabase(t)
	var subscriptions []models.Subcription
	for i := 0; i < 2; i++ {
		entry := models.Node{Name: fmt.Sprintf("node-%d", i), Link: fmt.Sprintf("ss://YWVzLTEyOC1nY206cGFzcw@example%d.test:443#Node%d", i, i)}
		if err := entry.Add(); err != nil {
			t.Fatal(err)
		}
		sub := models.Subcription{Name: fmt.Sprintf("subscription-%d", i), Nodes: []models.Node{entry}}
		if err := sub.Add(); err != nil {
			t.Fatal(err)
		}
		subscriptions = append(subscriptions, sub)
	}
	r := gin.New()
	r.GET("/c/", GetClient)
	var workers sync.WaitGroup
	for i := 0; i < 40; i++ {
		sub := subscriptions[i%2]
		workers.Add(1)
		go func() {
			defer workers.Done()
			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest("GET", "/c/?token="+sub.Token, nil))
			if response.Code != 200 || strings.TrimSpace(node.Base64Decode(response.Body.String())) != sub.Nodes[0].Link {
				t.Errorf("subscription isolation failed: %s", response.Body.String())
			}
		}()
	}
	workers.Wait()
	digest := md5.Sum([]byte(subscriptions[0].Name))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest("GET", "/c/?token="+hex.EncodeToString(digest[:]), nil))
	if response.Code != 401 {
		t.Fatal("old predictable link still accepted")
	}
}

func TestHTTPProxyIsNotFetchedAsSubscription(t *testing.T) {
	testDatabase(t)
	entry := models.Node{Name: "proxy", Link: "http://username:secret@127.0.0.1:1#Proxy", SourceType: "auto"}
	entry.Add()
	sub := models.Subcription{Name: "proxy-sub", Nodes: []models.Node{entry}}
	sub.Add()
	r := gin.New()
	r.GET("/c/", GetClient)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest("GET", "/c/?token="+sub.Token, nil))
	if response.Code != 200 || !strings.Contains(node.Base64Decode(response.Body.String()), entry.Link) {
		t.Fatal(response.Body.String())
	}
}

func TestLegacySchemaUpgradeWithMultipleSubscriptions(t *testing.T) {
	testDatabase(t)
	for _, err := range []error{
		models.DB.Migrator().DropIndex(&models.Subcription{}, "idx_subcriptions_token"),
		models.DB.Migrator().DropColumn(&models.Subcription{}, "Token"),
		models.DB.Migrator().DropColumn(&models.User{}, "SessionVersion"),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := models.DB.Omit("SessionVersion").Create(&models.User{Username: "old-admin", Password: "old-password"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first", "second"} {
		if err := models.DB.Omit("Token").Create(&models.Subcription{Name: name, Config: "{\"udp\":true}"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := models.DB.AutoMigrate(&models.User{}, &models.Subcription{}); err != nil {
		t.Fatal(err)
	}
	if err := models.MigrateSecurity(models.DB); err != nil {
		t.Fatal(err)
	}
	var subscriptions []models.Subcription
	if err := models.DB.Find(&subscriptions).Error; err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 2 || subscriptions[0].Token == subscriptions[1].Token {
		t.Fatal("legacy subscriptions were lost or given the same token")
	}
	for _, sub := range subscriptions {
		if len(sub.Token) != 64 || sub.Config != "{\"udp\":true}" {
			t.Fatal("legacy subscription data changed")
		}
	}
	if err := (&models.User{Username: "old-admin", Password: "old-password"}).Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidNodeDoesNotLogSecretURL(t *testing.T) {
	var output bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previousOutput)
	r := gin.New()
	r.POST("/node", NodeAdd)
	form := url.Values{"link": {"http://user:secret-password@host:invalid"}}
	request := httptest.NewRequest("POST", "/node", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != 400 || strings.Contains(output.String(), "secret-password") {
		t.Fatal("invalid node credential appeared in logs")
	}
}
