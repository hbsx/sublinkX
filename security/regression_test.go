package security

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFailureWindowDoesNotLoseEventsAcrossBoundary(t *testing.T) {
	s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}}
	base := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	for _, seconds := range []int{0, 298, 299, 301, 302, 303} {
		s.recordAt(testRequest(), "admin", false, "wrong", base.Add(time.Duration(seconds)*time.Second))
	}
	if len(s.queue) != 1 {
		t.Fatal("five recent failures did not generate an alert")
	}
	if note := <-s.queue; note.Failures != 5 {
		t.Fatal("incorrect failure count")
	}
}

func TestContinuedFailuresAlertOncePerFiveMinutes(t *testing.T) {
	s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}}
	base := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	for minute := 0; minute < 15; minute++ {
		s.recordAt(testRequest(), "admin", false, "wrong", base.Add(time.Duration(minute)*time.Minute))
		want := 0
		if minute >= 4 {
			want = 1 + (minute-4)/5
		}
		if len(s.queue) != want {
			t.Fatalf("minute %d: got %d alerts, want %d", minute, len(s.queue), want)
		}
	}
}

func TestExpiredFailuresCannotTriggerAlert(t *testing.T) {
	s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}}
	base := time.Now()
	for i := 0; i < 5; i++ {
		s.recordAt(testRequest(), "admin", false, "wrong", base.Add(time.Duration(i)*2*time.Minute))
	}
	if len(s.queue) != 0 {
		t.Fatal("counted failures outside the sliding window")
	}
}

func TestFullQueueDoesNotStartFailureCooldown(t *testing.T) {
	s := &Service{queue: make(chan notification, 1), config: Config{Enabled: true}}
	base := time.Now()
	s.recordAt(testRequest(), "admin", true, "ok", base)
	for i := 0; i < 5; i++ {
		s.recordAt(testRequest(), "admin", false, "wrong", base)
	}
	if !s.lastFail.IsZero() || s.status.Dropped != 1 {
		t.Fatal("unsent alert started cooldown")
	}
	<-s.queue
	s.recordAt(testRequest(), "admin", false, "wrong", base.Add(time.Second))
	if len(s.queue) != 1 {
		t.Fatal("failed alert was suppressed after queue recovered")
	}
}

func TestUnchangedSettingsPreserveQueueAndAvoidRepeatWrites(t *testing.T) {
	config := Config{Enabled: true, BotToken: "123456789:abcdefghijklmnopqrstuvwxyz_123456789", ChatID: "123456"}
	s := &Service{queue: make(chan notification, 32), config: config, configPath: filepath.Join(t.TempDir(), "settings.json")}
	s.Record(testRequest(), "admin", true, "ok")
	if err := s.Configure(config); err != nil {
		t.Fatal(err)
	}
	if s.status.Generation != 0 {
		t.Fatal("initial save invalidated an unchanged queued notification")
	}
	if _, err := os.Stat(s.configPath); err != nil {
		t.Fatal("initial save did not persist env-provided settings")
	}
	old := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(s.configPath, old, old); err != nil {
		t.Fatal(err)
	}
	config.BotToken = "" // UI deliberately omits the stored secret.
	if err := s.Configure(config); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(s.configPath)
	if !info.ModTime().Equal(old) || s.status.Generation != 0 {
		t.Fatal("unchanged save wrote to disk or invalidated queue")
	}
	note := <-s.queue
	if note.Generation != s.status.Generation {
		t.Fatal("login notification became stale")
	}
}

func TestCanceledQueueItemsAreCounted(t *testing.T) {
	s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}, status: Status{Generation: 2}}
	s.queue <- notification{Generation: 1}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.run(ctx)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if s.Snapshot().Notifications.Canceled == 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("discarded notification was invisible in counters")
}

func TestInFlightOldErrorDoesNotOverwriteNewConfigurationStatus(t *testing.T) {
	s := &Service{queue: make(chan notification, 1), config: Config{Enabled: true}, configPath: filepath.Join(t.TempDir(), "settings.json")}
	started, release := make(chan struct{}), make(chan struct{})
	s.client = func(Config) *http.Client {
		return &http.Client{Transport: roundTripper(func(*http.Request) (*http.Response, error) {
			close(started)
			<-release
			return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader(`{"ok":false}`))}, nil
		})}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.queue <- notification{Config: Config{BotToken: "old-token", ChatID: "123456"}}
	go s.run(ctx)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("send did not start")
	}
	if err := s.Configure(Config{Enabled: true, BotToken: "123456789:abcdefghijklmnopqrstuvwxyz_123456789", ChatID: "123456"}); err != nil {
		close(release)
		t.Fatal(err)
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status := s.Snapshot().Notifications
		if status.Failed == 1 {
			if status.LastError != "" {
				t.Fatal("old credentials error overwrote the new settings status")
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("in-flight notification did not finish")
}
