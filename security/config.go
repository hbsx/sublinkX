package security

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config is persisted only when an administrator explicitly saves settings.
// Never return Config in an API response; View deliberately excludes the token.
type Config struct {
	Enabled  bool   `json:"enabled"`
	BotToken string `json:"botToken"`
	ChatID   string `json:"chatId"`
	ProxyURL string `json:"proxyUrl"`
}

type View struct {
	Enabled            bool   `json:"enabled"`
	BotTokenConfigured bool   `json:"botTokenConfigured"`
	ChatID             string `json:"chatId"`
	ProxyURL           string `json:"proxyUrl"`
}

type Status struct {
	Sent       uint64    `json:"sent"`
	Failed     uint64    `json:"failed"`
	Dropped    uint64    `json:"dropped"`
	LastSentAt time.Time `json:"lastSentAt"`
	LastError  string    `json:"lastError"`
	Generation uint64    `json:"-"`
}

var tokenPattern = regexp.MustCompile(`^[0-9]{5,20}:[A-Za-z0-9_-]{20,100}$`)
var chatPattern = regexp.MustCompile(`^-?[0-9]{1,20}$`)

func validate(config Config) error {
	if config.BotToken != "" && !tokenPattern.MatchString(config.BotToken) {
		return errors.New("机器人 Token 格式错误")
	}
	if config.ChatID != "" && !chatPattern.MatchString(config.ChatID) {
		return errors.New("Chat ID 格式错误")
	}
	if config.Enabled && (config.BotToken == "" || config.ChatID == "") {
		return errors.New("请填写机器人 Token 和 Chat ID")
	}
	if config.ProxyURL != "" {
		u, err := url.Parse(config.ProxyURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return errors.New("代理地址应为不带账号密码的 HTTP/HTTPS 代理地址")
		}
	}
	return nil
}

func New(ctx context.Context, path string, getenv func(string) string) (*Service, error) {
	resolver, err := NewIPResolver(getenv("SUBLINK_LOGIN_TRUSTED_PROXIES"), getenv("SUBLINK_LOGIN_IP_HEADER"))
	if err != nil {
		return nil, err
	}
	config := Config{BotToken: getenv("SUBLINK_LOGIN_TG_BOT_TOKEN"), ChatID: getenv("SUBLINK_LOGIN_TG_CHAT_ID"), ProxyURL: getenv("SUBLINK_LOGIN_TG_PROXY_URL"), Enabled: getenv("SUBLINK_LOGIN_TG_ENABLED") == "1"}
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		if info, statErr := f.Stat(); statErr != nil || info.Size() > 8192 {
			return nil, errors.New("登录提醒配置过大或无法读取")
		}
		decoder := json.NewDecoder(io.LimitReader(f, 8193))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&config) != nil {
			return nil, errors.New("登录提醒配置无法读取")
		}
		var extra interface{}
		if decoder.Decode(&extra) != io.EOF {
			return nil, errors.New("登录提醒配置格式错误")
		}
	} else if !os.IsNotExist(err) {
		return nil, errors.New("登录提醒配置无法读取")
	}
	if err := validate(config); err != nil {
		return nil, err
	}
	s := &Service{startedAt: time.Now(), resolver: resolver, config: config, configPath: path, queue: make(chan notification, 32), client: telegramClient}
	go s.run(ctx)
	return s, nil
}

func (s *Service) View() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	return View{Enabled: s.config.Enabled, BotTokenConfigured: s.config.BotToken != "", ChatID: s.config.ChatID, ProxyURL: s.config.ProxyURL}
}

// A blank botToken preserves the existing secret, including secrets from env.
func (s *Service) Configure(next Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if next.BotToken == "" {
		next.BotToken = s.config.BotToken
	}
	next.ChatID, next.ProxyURL = strings.TrimSpace(next.ChatID), strings.TrimSpace(next.ProxyURL)
	if err := validate(next); err != nil {
		return err
	}
	data, err := json.Marshal(next)
	if err != nil {
		return errors.New("登录提醒配置无法保存")
	}
	f, err := os.CreateTemp(filepath.Dir(s.configPath), ".login-security-*")
	if err != nil {
		return errors.New("登录提醒配置无法保存")
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("登录提醒配置无法保存")
	}
	if err := os.Rename(f.Name(), s.configPath); err != nil {
		return errors.New("登录提醒配置无法保存")
	}
	s.config = next
	s.status.Generation++ // Discard queued notifications prepared with old credentials/settings.
	s.status.LastError = ""
	return nil
}

func (s *Service) TestNotification() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.config.Enabled {
		return false
	}
	return s.enqueueLocked(notification{Config: s.config, Generation: s.status.Generation, Test: true})
}

func telegramClient(config Config) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil // Never inherit unrelated system proxy settings.
	if config.ProxyURL != "" {
		proxy, _ := url.Parse(config.ProxyURL)
		transport.Proxy = http.ProxyURL(proxy)
	}
	return &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
