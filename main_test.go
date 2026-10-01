package main

import (
	"os"
	"testing"

	"sublink/api"
	"sublink/middlewares"
	"sublink/models"
)

func changeTestDirectory(t *testing.T) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Error(err)
		}
	})
}

func TestFirstStartAndRestartKeepLoginTokenValid(t *testing.T) {
	changeTestDirectory(t)
	previousSecret := middlewares.Secret
	t.Cleanup(func() { middlewares.Secret = previousSecret })

	first, err := initializeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(first.JwtSecret) != 64 {
		t.Fatal("fresh installations must generate a 32-byte random secret")
	}
	token, err := api.GetToken("admin")
	if err != nil {
		t.Fatal(err)
	}
	second, err := initializeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if second.JwtSecret != first.JwtSecret {
		t.Fatal("reinitializing changed the persisted JWT secret")
	}
	if _, err := middlewares.ParseToken(token); err != nil {
		t.Fatalf("login token did not survive reinitialization: %v", err)
	}
	if err := models.SetConfig(models.Config{Port: 8123}); err != nil {
		t.Fatal(err)
	}
	updated, err := models.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if updated.Port != 8123 || updated.JwtSecret != first.JwtSecret {
		t.Fatal("changing the port lost the existing configuration")
	}
}

func TestStartupRejectsInvalidConfigurationWithoutReplacingIt(t *testing.T) {
	for name, content := range map[string]string{
		"malformed":    "port: [",
		"empty_secret": "port: 8000\nexpire_days: 14\njwt_secret: ''\n",
		"bad_port":     "port: 70000\nexpire_days: 14\njwt_secret: existing-secret\n",
		"bad_expiry":   "port: 8000\nexpire_days: 0\njwt_secret: existing-secret\n",
	} {
		t.Run(name, func(t *testing.T) {
			changeTestDirectory(t)
			if err := os.Mkdir("db", 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("db/config.yaml", []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := initializeConfig(); err == nil {
				t.Fatal("startup accepted invalid configuration")
			}
			actual, err := os.ReadFile("db/config.yaml")
			if err != nil || string(actual) != content {
				t.Fatal("startup replaced the existing configuration")
			}
		})
	}
}
