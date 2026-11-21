package api

import (
	"net/http"
	"sublink/security"

	"github.com/gin-gonic/gin"
)

// Assigned once during startup; never replace while requests are running.
var LoginAudit *security.Service

func loginAuditAvailable(c *gin.Context) bool {
	c.Header("Cache-Control", "no-store")
	if LoginAudit == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "00001", "msg": "登录记录服务尚未启动"})
		return false
	}
	return true
}

func LoginHistory(c *gin.Context) {
	if loginAuditAvailable(c) {
		c.JSON(http.StatusOK, gin.H{"code": "00000", "data": LoginAudit.Snapshot()})
	}
}

func LoginNotificationConfig(c *gin.Context) {
	if loginAuditAvailable(c) {
		c.JSON(http.StatusOK, gin.H{"code": "00000", "data": LoginAudit.View()})
	}
}

func SaveLoginNotificationConfig(c *gin.Context) {
	if !loginAuditAvailable(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var next security.Config
	if c.ShouldBindJSON(&next) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "00001", "msg": "提醒配置格式错误"})
		return
	}
	if err := LoginAudit.Configure(next); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "00001", "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "00000", "data": LoginAudit.View(), "msg": "设置已保存"})
}

func TestLoginNotification(c *gin.Context) {
	if !loginAuditAvailable(c) {
		return
	}
	if !LoginAudit.TestNotification() {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": "00001", "msg": "请先启用并保存提醒配置，或稍后再试"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": "00000", "msg": "测试提醒已排队，请查看 Telegram；发送结果可刷新登录记录查看"})
}
