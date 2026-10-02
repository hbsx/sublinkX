package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
	"sublink/models"
	"sublink/node"
	"sublink/utils"
	"time"
)

func GetClient(c *gin.Context) {
	token := strings.ToLower(c.Query("token"))
	if len(token) != 64 {
		c.String(401, "订阅凭据无效，请从后台重新复制链接")
		return
	}
	sub := &models.Subcription{}
	if err := sub.FindByToken(token); err != nil {
		c.String(401, "订阅凭据无效")
		return
	}
	c.Set("subscription", sub)
	c.Set("subname", sub.Name)
	c.Set("subscription_id", sub.ID)
	client := strings.ToLower(c.Query("client"))
	if client == "" {
		agent := strings.ToLower(c.GetHeader("User-Agent"))
		switch {
		case strings.Contains(agent, "clash"):
			client = "clash"
		case strings.Contains(agent, "surge"):
			client = "surge"
		default:
			client = "v2ray"
		}
	}
	switch client {
	case "clash":
		GetClash(c)
	case "surge":
		GetSurge(c)
	case "v2ray":
		GetV2ray(c)
	default:
		c.String(400, "未知客户端")
	}
}

func subscriptionURLs(c *gin.Context) (*models.Subcription, []string, error) {
	value, ok := c.Get("subscription")
	sub, valid := value.(*models.Subcription)
	if !ok || !valid {
		return nil, nil, fmt.Errorf("未选择订阅")
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	var links []string
	for _, entry := range sub.Nodes {
		for _, link := range node.SplitLinks(entry.Link) {
			link = strings.TrimSpace(link)
			if link == "" {
				continue
			}
			if utils.IsRemoteSubscription(link, entry.SourceType) {
				body, err := utils.Fetch(ctx, link, 8<<20)
				if err != nil {
					return nil, nil, err
				}
				remote, err := node.DecodeSubscription(string(body))
				if err != nil {
					return nil, nil, err
				}
				links = append(links, remote...)
			} else {
				if _, err := node.NodeName(link); err != nil {
					return nil, nil, err
				}
				links = append(links, link)
			}
		}
	}
	return sub, links, nil
}

func subscriptionError(c *gin.Context) {
	c.String(502, "订阅生成失败，请检查远程来源和模板配置")
}

func sendSubscription(c *gin.Context, sub *models.Subcription, extension, content string) {
	filename := url.QueryEscape(sub.Name + "." + extension)
	c.Header("Content-Disposition", "inline; filename*=utf-8''"+filename)
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
}

func GetV2ray(c *gin.Context) {
	sub, links, err := subscriptionURLs(c)
	if err != nil {
		subscriptionError(c)
		return
	}
	sendSubscription(c, sub, "txt", node.Base64Encode(strings.Join(links, "\n")+"\n"))
}

func GetClash(c *gin.Context) {
	sub, links, err := subscriptionURLs(c)
	if err != nil {
		subscriptionError(c)
		return
	}
	var config node.SqlConfig
	if json.Unmarshal([]byte(sub.Config), &config) != nil {
		subscriptionError(c)
		return
	}
	result, err := node.EncodeClash(links, config)
	if err != nil {
		subscriptionError(c)
		return
	}
	sendSubscription(c, sub, "yaml", string(result))
}

func GetSurge(c *gin.Context) {
	sub, links, err := subscriptionURLs(c)
	if err != nil {
		subscriptionError(c)
		return
	}
	var config node.SqlConfig
	if json.Unmarshal([]byte(sub.Config), &config) != nil {
		subscriptionError(c)
		return
	}
	result, err := node.EncodeSurge(links, config)
	if err != nil {
		subscriptionError(c)
		return
	}
	if !strings.Contains(result, "#!MANAGED-CONFIG") {
		scheme := requestScheme(c)
		result = fmt.Sprintf("#!MANAGED-CONFIG %s://%s%s interval=86400 strict=false\n%s",
			scheme, c.Request.Host, c.Request.URL.RequestURI(), result)
	}
	sendSubscription(c, sub, "conf", result)
}

func requestScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}
	if value := strings.ToLower(strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])); value == "https" {
		return "https"
	}
	return "http"
}
