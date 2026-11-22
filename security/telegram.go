package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type notification struct {
	Entry      Login
	Config     Config
	Generation uint64
	Failures   int
	Test       bool
}

func (s *Service) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case note := <-s.queue:
			s.mu.Lock()
			valid := s.config.Enabled && note.Generation == s.status.Generation
			if !valid {
				s.status.Canceled++
			}
			s.mu.Unlock()
			if !valid {
				continue
			}
			errText := s.deliver(ctx, note)
			s.mu.Lock()
			if errText == "" {
				s.status.Sent++
				s.status.LastSentAt = time.Now()
				if note.Generation == s.status.Generation {
					s.status.LastError = ""
				}
			} else {
				s.status.Failed++
				if note.Generation == s.status.Generation {
					s.status.LastError = errText
				}
			}
			s.mu.Unlock()
		}
	}
}

func message(note notification) string {
	if note.Test {
		return "SublinkX 登录提醒测试：Telegram 连接正常。"
	}
	e := note.Entry
	title := "SublinkX 登录成功"
	if !e.Success {
		title = fmt.Sprintf("SublinkX 登录异常：5 分钟内至少 %d 次失败", note.Failures)
	}
	return fmt.Sprintf("%s\n时间：%s\n账号：%s\n来源 IP：%s\nIP 来源：%s\n设备：%s", title, e.Time.Format(time.RFC3339), e.Username, e.IP, e.IPSource, e.UserAgent)
}

func (s *Service) deliver(ctx context.Context, note notification) string {
	data, _ := json.Marshal(map[string]interface{}{"chat_id": note.Config.ChatID, "text": message(note), "disable_web_page_preview": true})
	request, err := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+note.Config.BotToken+"/sendMessage", strings.NewReader(string(data)))
	if err != nil {
		return "Telegram 请求创建失败"
	}
	request.Header.Set("Content-Type", "application/json")
	client := s.client(note.Config)
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		return "Telegram 连接失败，请检查代理和网络"
	} // Do not expose errors containing the token URL.
	defer response.Body.Close()
	var result struct {
		OK bool `json:"ok"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 16384)).Decode(&result) != nil || response.StatusCode != http.StatusOK || !result.OK {
		return "Telegram 发送失败，请检查 Token、Chat ID 或发送频率"
	}
	return ""
}
