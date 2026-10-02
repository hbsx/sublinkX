package api

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sublink/models"
	"sublink/node"
	"sublink/utils"

	"github.com/gin-gonic/gin"
)

func DocodeNodeName(nd *models.Node) (models.Node, error) {
	links := node.SplitLinks(nd.Link)
	if len(links) == 0 {
		return *nd, fmt.Errorf("节点链接不能为空")
	}
	for _, link := range links {
		if utils.IsRemoteSubscription(link, nd.SourceType) {
			u, err := url.Parse(link)
			if err != nil || u.Hostname() == "" {
				return *nd, fmt.Errorf("订阅地址无效")
			}
			if nd.Name == "" {
				nd.Name = u.Hostname()
			}
			continue
		}
		if nd.SourceType == "subscription" {
			return *nd, fmt.Errorf("订阅地址必须使用 HTTP 或 HTTPS")
		}
		name, err := node.NodeName(link)
		if err != nil {
			return *nd, err
		}
		if nd.Name == "" {
			nd.Name = name
		}
	}
	return *nd, nil
}
func NodeUpdadte(c *gin.Context) {
	// var node models.Node
	NewName := c.PostForm("name")
	Newlink := c.PostForm("link")
	sourceType := c.PostForm("source_type")
	if sourceType != "" && sourceType != "auto" && sourceType != "proxy" && sourceType != "subscription" {
		c.JSON(400, gin.H{"msg": "链接类型无效"})
		return
	}
	id := c.PostForm("id")
	group := c.PostForm("group")        // 分组
	groups := strings.Split(group, ",") // 分组列表
	index, err := strconv.Atoi(id)
	if err != nil || index <= 0 {
		c.JSON(400, gin.H{
			"msg": "id 不能为空或者格式不正确",
		})
		return

	}
	if NewName == "" || Newlink == "" {
		c.JSON(400, gin.H{
			"msg": "节点名称 or 备注不能为空",
		})
		return
	}
	OldNode := &models.Node{
		ID: index,
	}
	NewNode := &models.Node{
		Name:       NewName,
		Link:       Newlink,
		SourceType: sourceType,
	}
	if _, err := DocodeNodeName(NewNode); err != nil {
		c.JSON(400, gin.H{"msg": "节点链接无效"})
		return
	}
	var gns []models.GroupNode
	if groups != nil || len(groups) > 0 {
		for _, g := range groups {
			TempGn := models.GroupNode{
				Name: strings.TrimSpace(g), // 去除分组名称两端空格
			}
			gns = append(gns, TempGn) // 生成分组列表
		}

	}
	err = OldNode.UpdateNodeAndGroups(NewNode, gns)
	if err != nil {
		c.JSON(400, gin.H{"msg": "更新节点失败"})
		return
	}

	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "更新成功",
	})
}

// 获取节点列表
func NodeGet(c *gin.Context) {
	var ns []models.Node
	ns, err := models.GetNodeList()
	if err != nil {
		c.JSON(500, gin.H{
			"msg": "node list error",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": ns,
		"msg":  "node get",
	})
}

// 获取分组列表
func GroupNodeGet(c *gin.Context) {
	var Gns []models.GroupNode
	Gns, err := models.GetGroupNodeList()
	var data []string
	for _, g := range Gns {
		data = append(data, g.Name)
	}
	if err != nil {
		c.JSON(400, gin.H{
			"msg": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": data,
		"msg":  "GroupNode get",
	})
}

// 设置关联分组
func GroupNodeSet(c *gin.Context) {
	n := models.Node{Name: c.PostForm("name")}
	if raw := c.PostForm("id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			c.JSON(400, gin.H{"msg": "节点 ID 无效"})
			return
		}
		n.ID = id
	}
	groups := groupForm(c.PostForm("group"))
	if err := n.UpdateGroup(groups); err != nil {
		c.JSON(400, gin.H{"msg": "更新关联分组失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "更新关联分组成功"})
}

func groupForm(raw string) []models.GroupNode {
	groups := []models.GroupNode{}
	for _, name := range strings.Split(raw, ",") {
		if name = strings.TrimSpace(name); name != "" {
			groups = append(groups, models.GroupNode{Name: name})
		}
	}
	return groups
}

// 添加节点
func NodeAdd(c *gin.Context) {
	var n models.Node
	link := c.PostForm("link")
	sourceType := c.PostForm("source_type")
	if sourceType != "" && sourceType != "auto" && sourceType != "proxy" && sourceType != "subscription" {
		c.JSON(400, gin.H{"msg": "链接类型无效"})
		return
	}
	name := c.PostForm("name")
	group := c.PostForm("group")
	n = models.Node{
		Name:       name,
		Link:       link,
		SourceType: sourceType,
	}
	if link == "" || !strings.Contains(link, "://") {
		c.JSON(400, gin.H{
			"msg": "link不能为空或者格式不正确,请检查链接是否包含协议头,例如 http:// 或 https://",
		})
		return
	}
	// 解码节点名称
	n, err := DocodeNodeName(&n)
	if err != nil {
		log.Println("解码节点名称错误")
		c.JSON(400, gin.H{
			"msg": "解码节点名称错误",
		})
		return
	}

	if err := n.AddWithGroups(groupForm(group)); err != nil {
		c.JSON(400, gin.H{"msg": "添加节点或分组失败"})
		return
	}

	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "添加成功",
	})
}

// 删除节点
func NodeDel(c *gin.Context) {
	var n models.Node
	id := c.Query("id")
	if id == "" {
		c.JSON(400, gin.H{
			"msg": "id 不能为空",
		})
		return
	}
	x, err := strconv.Atoi(id)
	if err != nil || x <= 0 {
		c.JSON(400, gin.H{"msg": "节点 ID 无效"})
		return
	}
	n.ID = x
	err = n.Del()
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "删除失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "删除成功",
	})
}

// 删除分组
func NodesGroup(c *gin.Context) {
	var gn models.GroupNode
	id := c.Query("id")
	if id == "" {
		c.JSON(400, gin.H{
			"msg": "id 不能为空",
		})
		return
	}
	x, err := strconv.Atoi(id)
	if err != nil || x <= 0 {
		c.JSON(400, gin.H{"msg": "分组 ID 无效"})
		return
	}
	gn.ID = x
	err = gn.Del()
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "删除失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "删除成功",
	})
}

// 节点统计
func NodesTotal(c *gin.Context) {
	var nodes []models.Node
	nodes, err := models.GetNodeList()
	count := len(nodes)
	if err != nil {
		c.JSON(500, gin.H{
			"msg": "获取不到节点统计",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": count,
		"msg":  "取得节点统计",
	})
}
