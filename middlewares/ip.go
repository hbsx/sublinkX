package middlewares

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sublink/models"
	"sublink/utils"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func GetIp(c *gin.Context) {
	c.Next()
	func() {
		subscriptionID, valid := c.Get("subscription_id")
		if !valid || c.Writer.Status() != 200 {
			return
		}
		ip := c.ClientIP()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		body, err := utils.Fetch(ctx, fmt.Sprintf("https://whois.pconline.com.cn/ipJson.jsp?ip=%s&json=true", ip), 64<<10)
		if err != nil {
			log.Println("获取IP信息失败:", err)
			return
		}
		utf8Body, _ := simplifiedchinese.GBK.NewDecoder().Bytes(body)
		type IpInfo struct {
			Addr string `json:"addr"`
			Ip   string `json:"ip"`
		}
		ipinfo := IpInfo{}
		err = json.Unmarshal(utf8Body, &ipinfo)
		if err != nil {
			log.Println("解析IP信息失败:", err)
			return
		}

		id, ok := subscriptionID.(int)
		if !ok || id <= 0 {
			return
		}
		sub := models.Subcription{ID: id}
		if err := sub.Find(); err != nil {
			return
		}

		var iplog models.SubLogs
		iplog.IP = ip
		// 查找是否存在该 IP 记录
		err = iplog.Find(sub.ID) // 这里 `iplog.Find` 内部会根据 `iplog.IP` 和 `sub.ID` 查找
		log.Println("查找IP日志记录结果:", err)

		// 如果没有找到记录，则创建新记录
		if err != nil {
			log.Println("未找到现有IP日志，将创建新记录。")
			newIplog := models.SubLogs{
				IP:            ip,
				Addr:          ipinfo.Addr,
				SubcriptionID: sub.ID,
				Date:          time.Now().Format("2006-01-02 15:04:05"),
				Count:         1,
			}
			err = newIplog.Add() // 使用 iplogs.go 中的 Add 方法
			if err != nil {
				log.Println("添加IP日志记录失败:", err)
				return
			}
			log.Println("成功添加新的IP日志记录。")
		} else {
			// 如果找到了记录，则更新访问次数和日期
			log.Println("找到现有IP日志，将更新记录。")
			iplog.Count++
			iplog.Date = time.Now().Format("2006-01-02 15:04:05")
			err = iplog.Update() // 使用 iplogs.go 中的 Update 方法
			if err != nil {
				log.Println("更新IP日志记录失败:", err)
				return
			}
			log.Println("成功更新IP日志记录。")
		}
	}()
}
