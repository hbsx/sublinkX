package security

import (
	"net"
	"testing"
	"time"
)

func TestOnlyPublicLoginsAffectHistoryAndNotifications(t *testing.T) {
	for _, ip := range []string{
		"192.168.1.160", "10.0.0.1", "172.16.0.1", "172.31.255.255", "127.0.0.1",
		"169.254.1.1", "100.64.0.1", "100.127.255.255", "0.1.2.3", "0.0.0.0",
		"198.18.0.1", "192.0.2.1", "198.51.100.1", "203.0.113.1", "192.0.0.8", "192.88.99.2",
		"224.0.0.1", "240.0.0.1", "255.255.255.255", "::", "::1", "fd00::1", "fc00::1",
		"fe80::1", "ff02::1", "2001:db8::1", "3fff::1", "100::1", "100:0:0:1::1",
		"64:ff9b:1::1", "2001:2::1", "5f00::1", "::ffff:192.168.1.160", "::ffff:100.64.0.1",
		"fe80::1%eth0", "unknown", "not-an-ip",
	} {
		t.Run(ip, func(t *testing.T) {
			s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}}
			s.Record(testRequest(), "external", true, "ok")
			before := s.Snapshot()
			request := testRequest()
			request.RemoteAddr = net.JoinHostPort(ip, "12345")
			s.Record(request, "local", true, "ok")
			for i := 0; i < 6; i++ {
				s.Record(request, "local", false, "wrong")
			}
			if s.next != 1 || len(s.Snapshot().List) != 1 || s.Snapshot().List[0] != before.List[0] || len(s.queue) != 1 || s.failCount != 0 || !s.lastFail.IsZero() {
				t.Fatal("non-public login changed audit history, queue or failure counters")
			}
		})
	}
	for _, ip := range []string{"1.1.1.1", "8.8.8.8", "100.63.255.255", "100.128.0.1", "172.15.255.255", "172.32.0.1", "192.0.0.9", "192.0.0.10", "2606:4700:4700::1111", "240e::1", "::ffff:8.8.8.8"} {
		t.Run(ip, func(t *testing.T) {
			s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}}
			request := testRequest()
			request.RemoteAddr = net.JoinHostPort(ip, "12345")
			s.Record(request, "external", true, "ok")
			for i := 0; i < 5; i++ {
				s.Record(request, "external", false, "wrong")
			}
			if len(s.Snapshot().List) != 6 || len(s.queue) != 2 || s.failCount != 5 {
				t.Fatal("public login lost history or notification")
			}
		})
	}
}

func TestPublicLoginFilterUsesTrustedVisitorAddress(t *testing.T) {
	resolver, err := NewIPResolver("172.17.0.1", "CF-Connecting-IP")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		peer, header string
		count        int
	}{
		{"172.17.0.1:12345", "8.8.8.8", 1},
		{"172.17.0.1:12345", "2606:4700:4700::1111", 1},
		{"172.17.0.1:12345", "192.168.1.10", 0},
		{"172.17.0.1:12345", "", 0},
		{"172.17.0.1:12345", "invalid", 0},
		{"192.168.1.10:12345", "8.8.8.8", 0},
		{"1.1.1.1:12345", "192.168.1.10", 1},
	} {
		t.Run(tc.peer+"/"+tc.header, func(t *testing.T) {
			s := &Service{resolver: resolver, queue: make(chan notification, 32), config: Config{Enabled: true}}
			request := testRequest()
			request.RemoteAddr = tc.peer
			request.Header.Set("CF-Connecting-IP", tc.header)
			s.Record(request, "admin", true, "ok")
			if len(s.Snapshot().List) != tc.count || len(s.queue) != tc.count {
				t.Fatal("filter did not use the resolved visitor address")
			}
		})
	}
}

func TestLocalFailuresCannotContributeToPublicFailureAlert(t *testing.T) {
	s := &Service{queue: make(chan notification, 32), config: Config{Enabled: true}}
	local := testRequest()
	local.RemoteAddr = "192.168.1.10:12345"
	for i := 0; i < 4; i++ {
		s.Record(testRequest(), "external", false, "wrong")
	}
	for i := 0; i < 10; i++ {
		s.Record(local, "local", false, "wrong")
	}
	if len(s.queue) != 0 || s.failCount != 4 || !s.lastFail.IsZero() {
		t.Fatal("local failures triggered a public failure alert")
	}
	s.Record(testRequest(), "external", false, "wrong")
	if len(s.queue) != 1 || s.lastFail.Equal(time.Time{}) {
		t.Fatal("fifth public failure did not alert")
	}
}
