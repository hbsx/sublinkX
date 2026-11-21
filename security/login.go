// Package security keeps login audit data in bounded memory, never in the database.
package security

import (
	"net/http"
	"sync"
	"time"
	"unicode"
)

const HistoryLimit = 20

type Login struct {
	ID        uint64    `json:"id"`
	Time      time.Time `json:"time"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	IPSource  string    `json:"ipSource"`
	UserAgent string    `json:"userAgent"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
}

type Snapshot struct {
	List          []Login   `json:"list"`
	Limit         int       `json:"limit"`
	StartedAt     time.Time `json:"startedAt"`
	Notifications Status    `json:"notifications"`
}

type Service struct {
	mu         sync.Mutex
	records    [HistoryLimit]Login
	next       uint64
	startedAt  time.Time
	resolver   *IPResolver
	config     Config
	configPath string
	status     Status
	failCount  int
	failStart  time.Time
	lastFail   time.Time
	queue      chan notification
	client     func(Config) *http.Client
}

func clean(value string, limit int) string {
	out := make([]rune, 0, limit)
	for _, r := range value {
		if !unicode.IsControl(r) && !unicode.Is(unicode.Cf, r) {
			out = append(out, r)
			if len(out) == limit {
				break
			}
		}
	}
	return string(out)
}

func (s *Service) Record(request *http.Request, username string, success bool, reason string) {
	ip, source := s.resolver.Resolve(request)
	entry := Login{Time: time.Now(), Username: clean(username, 64), IP: ip,
		IPSource: source, UserAgent: clean(request.UserAgent(), 180), Success: success, Reason: reason}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	entry.ID = s.next
	s.records[(s.next-1)%HistoryLimit] = entry
	if !s.config.Enabled {
		return
	}
	note := notification{Entry: entry, Config: s.config, Generation: s.status.Generation}
	if !success {
		if s.failStart.IsZero() || entry.Time.Sub(s.failStart) >= 5*time.Minute {
			s.failStart, s.failCount = entry.Time, 0
		}
		s.failCount++
		if s.failCount < 5 || (!s.lastFail.IsZero() && entry.Time.Sub(s.lastFail) < 5*time.Minute) {
			return
		}
		note.Failures = s.failCount
		s.failCount, s.lastFail, s.failStart = 0, entry.Time, entry.Time
	}
	s.enqueueLocked(note)
}

func (s *Service) enqueueLocked(note notification) bool {
	select {
	case s.queue <- note:
		return true
	default:
		s.status.Dropped++
		return false
	}
}

func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.next
	if n > HistoryLimit {
		n = HistoryLimit
	}
	entries := make([]Login, 0, n)
	for i := uint64(0); i < n; i++ {
		entries = append(entries, s.records[(s.next-i-1)%HistoryLimit])
	}
	return Snapshot{List: entries, Limit: HistoryLimit, StartedAt: s.startedAt, Notifications: s.status}
}
