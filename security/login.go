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
	mu          sync.Mutex
	records     [HistoryLimit]Login
	next        uint64
	startedAt   time.Time
	resolver    *IPResolver
	config      Config
	configSaved bool
	configPath  string
	status      Status
	failTimes   [5]time.Time
	failNext    int
	failCount   int
	lastFail    time.Time
	queue       chan notification
	client      func(Config) *http.Client
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
	s.recordAt(request, username, success, reason, time.Now())
}

func (s *Service) recordAt(request *http.Request, username string, success bool, reason string, now time.Time) {
	ip, source := s.resolver.Resolve(request)
	entry := Login{Time: now, Username: clean(username, 64), IP: ip,
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
		// Keep only the last five timestamps: a true sliding window in fixed memory.
		s.failTimes[s.failNext] = now
		s.failNext = (s.failNext + 1) % len(s.failTimes)
		if s.failCount < len(s.failTimes) {
			s.failCount++
		}
		if s.failCount < len(s.failTimes) || now.Sub(s.failTimes[s.failNext]) > 5*time.Minute || (!s.lastFail.IsZero() && now.Sub(s.lastFail) < 5*time.Minute) {
			return
		}
		note.Failures = len(s.failTimes)
		// A full queue must not suppress the next opportunity for five minutes.
		if s.enqueueLocked(note) {
			s.lastFail = now
		}
		return
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
