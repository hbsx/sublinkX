package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchTimeoutStatusAndSizeLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wait":
			<-r.Context().Done()
		case "/bad":
			w.WriteHeader(403)
		default:
			w.Write([]byte("too much content"))
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := Fetch(ctx, server.URL+"/wait", 1024); err == nil || time.Since(start) > time.Second {
		t.Fatal("request failed to time out")
	}
	if _, err := Fetch(context.Background(), server.URL+"/bad", 1024); err == nil {
		t.Fatal("HTTP failure accepted")
	}
	if _, err := Fetch(context.Background(), server.URL, 2); err == nil {
		t.Fatal("oversized content accepted")
	}
}
