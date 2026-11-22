package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testService(t *testing.T) *Service {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s, err := New(ctx, filepath.Join(t.TempDir(), "login.json"), func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func testRequest() *http.Request {
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "1.1.1.1:443"
	return r
}

func TestHistoryIsBoundedOrderedAndNotPersisted(t *testing.T) {
	s := testService(t)
	for i := 1; i <= 25; i++ {
		s.Record(testRequest(), fmt.Sprint(i), i%2 == 0, "test")
	}
	snap := s.Snapshot()
	if len(snap.List) != 20 || snap.List[0].Username != "25" || snap.List[19].Username != "6" {
		t.Fatalf("bad ring: %+v", snap.List)
	}
	snap.List[0].IP = "modified"
	if s.Snapshot().List[0].IP == "modified" {
		t.Fatal("snapshot aliases stored data")
	}
	if _, err := os.Stat(s.configPath); !os.IsNotExist(err) {
		t.Fatal("recording wrote a file")
	}
	restarted := testService(t)
	if len(restarted.Snapshot().List) != 0 {
		t.Fatal("history survived restart")
	}
}

func TestConcurrentRecordsAreBounded(t *testing.T) {
	s := testService(t)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); s.Record(testRequest(), "admin", true, "ok"); s.Snapshot() }()
	}
	wg.Wait()
	if snap := s.Snapshot(); len(snap.List) != 20 || snap.List[0].ID != 100 {
		t.Fatal("lost concurrent events")
	}
}

func TestForwardedIPCannotBeSpoofedByUntrustedPeer(t *testing.T) {
	r, err := NewIPResolver("172.17.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	req := testRequest()
	req.Header.Set("CF-Connecting-IP", "2001:db8::7")
	if ip, _ := r.Resolve(req); ip != "1.1.1.1" {
		t.Fatal("trusted spoofed header")
	}
	req.RemoteAddr = "172.17.0.1:51000"
	if ip, source := r.Resolve(req); ip != "2001:db8::7" || source != "CF-Connecting-IP" {
		t.Fatal("lost trusted header", ip, source)
	}
	for _, invalid := range []string{"not-an-ip", "1.2.3.4,5.6.7.8", "fe80::1%eth0"} {
		req.Header.Set("CF-Connecting-IP", invalid)
		if ip, _ := r.Resolve(req); ip != "172.17.0.1" {
			t.Fatal("accepted malformed header")
		}
	}
	req.Header.Set("CF-Connecting-IP", "1.2.3.4")
	req.Header.Add("CF-Connecting-IP", "5.6.7.8")
	if ip, _ := r.Resolve(req); ip != "172.17.0.1" {
		t.Fatal("accepted duplicate headers")
	}
	if _, err := NewIPResolver("0.0.0.0/0", ""); err == nil {
		t.Fatal("trusted all peers")
	}
}

func TestXFFUsesRightmostUntrustedHop(t *testing.T) {
	r, _ := NewIPResolver("127.0.0.1,172.17.0.1", "X-Forwarded-For")
	req := testRequest()
	req.RemoteAddr = "127.0.0.1:3000"
	req.Header.Set("X-Forwarded-For", "8.8.8.8, 1.1.1.1, 172.17.0.1")
	if ip, _ := r.Resolve(req); ip != "1.1.1.1" {
		t.Fatal("spoofed leftmost IP won", ip)
	}
}

func TestAlertQueueAndFailedLoginAggregationAreBounded(t *testing.T) {
	// No worker: inspect notifications deterministically without any network calls.
	s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}}
	for i := 0; i < 4; i++ {
		s.Record(testRequest(), "admin", false, "wrong")
	}
	if len(s.queue) != 0 {
		t.Fatal("unaggregated failures")
	}
	s.Record(testRequest(), "admin", false, "wrong")
	if note := <-s.queue; note.Failures != 5 || note.Entry.Success {
		t.Fatal("bad failure alert")
	}
	for i := 0; i < 100; i++ {
		s.Record(testRequest(), "admin", false, "wrong")
	}
	if len(s.queue) != 0 {
		t.Fatal("failure alert flood")
	}
	for i := 0; i < 40; i++ {
		s.Record(testRequest(), "admin", true, "ok")
	}
	if len(s.queue) != 32 || s.Snapshot().Notifications.Dropped != 8 {
		t.Fatal("unbounded or blocking queue")
	}
}

func TestConfigIsWriteOnlyAndBlankTokenPreservesSecret(t *testing.T) {
	s := testService(t)
	secret := "123456789:abcdefghijklmnopqrstuvwxyz_123456789"
	if err := s.Configure(Config{BotToken: secret, ChatID: "123456"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Configure(Config{Enabled: true, ChatID: "123456"}); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(s.View())
	if strings.Contains(string(data), secret) || !s.View().BotTokenConfigured {
		t.Fatal("secret was exposed or cleared")
	}
	config, err := os.ReadFile(s.configPath)
	if err != nil || !strings.Contains(string(config), secret) {
		t.Fatal("settings not persisted")
	}
	if os.PathSeparator != '\\' {
		info, _ := os.Stat(s.configPath)
		if info.Mode().Perm() != 0600 {
			t.Fatal("unsafe config permissions")
		}
	}
	before := s.View()
	if err := s.Configure(Config{Enabled: true, ChatID: "123456", ProxyURL: "http://user:password@proxy.test:7890"}); err == nil {
		t.Fatal("accepted exposed proxy credentials")
	}
	if before != s.View() {
		t.Fatal("failed update changed settings")
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTelegramPayloadAndErrorNeverExposeSecret(t *testing.T) {
	s := testService(t)
	note := notification{Config: Config{BotToken: "123456789:abcdefghijklmnopqrstuvwxyz_123456789", ChatID: "123456"}, Entry: Login{Time: time.Now(), Username: "admin", IP: "1.1.1.1", Success: true}}
	s.client = func(Config) *http.Client {
		return &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
			if r.URL.Host != "api.telegram.org" || r.Method != "POST" {
				t.Fatal("wrong destination")
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "1.1.1.1") || strings.Contains(string(body), note.Config.BotToken) {
				t.Fatal("bad alert payload")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})}
	}
	if result := s.deliver(context.Background(), note); result != "" {
		t.Fatal(result)
	}
	s.client = func(Config) *http.Client {
		return &http.Client{Transport: roundTripper(func(*http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("secret URL %s", note.Config.BotToken)
		})}
	}
	if result := s.deliver(context.Background(), note); result == "" || strings.Contains(result, note.Config.BotToken) {
		t.Fatal("leaked error", result)
	}
}

func TestDisableDiscardsQueuedMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Service{queue: make(chan notification, 32)}
	s.queue <- notification{Config: Config{Enabled: true}, Test: true}
	s.client = func(Config) *http.Client { t.Error("disabled notification sent"); return http.DefaultClient }
	go s.run(ctx)
	deadline := time.Now().Add(time.Second)
	for len(s.queue) != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(s.queue) != 0 {
		t.Fatal("queue not consumed")
	}
}
