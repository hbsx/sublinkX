// api/subcription.go

package api

import (
	// 导入 json 包，用于解析 config 字符串

	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sublink/models" // 导入 models 包

	"github.com/gin-gonic/gin"
)

func SubTotal(c *gin.Context) {
	var Sub models.Subcription
	subs, err := Sub.List()
	count := len(subs)
	if err != nil {
		c.JSON(500, gin.H{
			"msg": "取得订阅总数失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": count,
		"msg":  "取得订阅总数",
	})
}

// 获取订阅列表
func SubGet(c *gin.Context) {
	var Sub models.Subcription
	Subs, err := Sub.List()
	if err != nil {
		c.JSON(500, gin.H{
			"msg": "node list error",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": Subs,
		"msg":  "node get",
	})
}

func selectedNodes(c *gin.Context) ([]models.Node, error) {
	nodes := []models.Node{}
	raw := c.PostForm("node_ids")
	if raw != "" {
		var ids []int
		if err := json.Unmarshal([]byte(raw), &ids); err != nil || len(ids) == 0 {
			return nil, fmt.Errorf("节点 ID 列表无效")
		}
		seen := map[int]bool{}
		for _, id := range ids {
			if id <= 0 {
				return nil, fmt.Errorf("节点 ID 无效")
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			var n models.Node
			if err := models.DB.First(&n, id).Error; err != nil {
				return nil, fmt.Errorf("节点不存在")
			}
			nodes = append(nodes, n)
		}
	} else {
		for _, name := range strings.Split(c.PostForm("nodes"), ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			var matches []models.Node
			if err := models.DB.Where("name = ?", name).Find(&matches).Error; err != nil {
				return nil, err
			}
			if len(matches) != 1 {
				return nil, fmt.Errorf("节点不存在或同名，请刷新页面后按 ID 选择")
			}
			nodes = append(nodes, matches[0])
		}
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("至少选择一个节点")
	}
	return nodes, nil
}
func subscriptionForm(c *gin.Context) (*models.Subcription, error) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		return nil, fmt.Errorf("订阅名称不能为空")
	}
	nodes, err := selectedNodes(c)
	if err != nil {
		return nil, err
	}
	config := c.PostForm("config")
	var settings models.SubscriptionConfig
	if json.Unmarshal([]byte(config), &settings) != nil {
		return nil, fmt.Errorf("订阅配置无效")
	}
	return &models.Subcription{Name: name, Config: config, Nodes: nodes}, nil
}
func SubAdd(c *gin.Context) {
	sub, err := subscriptionForm(c)
	if err == nil {
		err = sub.Add()
	}
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "添加订阅成功"})
}
func SubUpdate(c *gin.Context) {
	sub, err := subscriptionForm(c)
	old := models.Subcription{Name: c.PostForm("oldname")}
	if raw := c.PostForm("id"); raw != "" {
		id, idErr := strconv.Atoi(raw)
		old.ID = id
		if idErr != nil || old.ID <= 0 {
			c.JSON(400, gin.H{"msg": "订阅 ID 无效"})
			return
		}
	}
	if err == nil {
		err = old.Update(sub)
	}
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "更新订阅成功"})
}

// 删除订阅 (无需修改)
func SubDel(c *gin.Context) {
	var sub models.Subcription
	id := c.Query("id")
	if id == "" {
		c.JSON(400, gin.H{
			"msg": "id 不能为空",
		})
		return
	}
	x, err := strconv.Atoi(id) // 增加错误检查
	if err != nil || x <= 0 {
		c.JSON(400, gin.H{
			"msg": "无效的订阅 ID",
		})
		return
	}
	sub.ID = x
	err = sub.Find()
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "查找订阅失败: " + err.Error(),
		})
		return
	}
	err = sub.Del()
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "删除订阅失败: " + err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "删除订阅成功",
	})
}
