package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sublink/models"

	"github.com/gin-gonic/gin"
)

const nodeTransferLimit = 5 << 20
const nodeTransferCount = 5000

type nodeBackupFile struct {
	Format  string              `json:"format"`
	Version int                 `json:"version"`
	Nodes   []models.NodeBackup `json:"nodes"`
}
type nodeImportRequest struct {
	Format  string `json:"format"`
	Content string `json:"content"`
	Confirm bool   `json:"confirm"`
}
type nodeImportRow struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func strictJSON(data string, target interface{}) error {
	d := json.NewDecoder(strings.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(interface{})); err != io.EOF {
		return fmt.Errorf("unexpected trailing JSON")
	}
	return nil
}

// NodeImport previews without writing; confirmation validates the entire file again.
func NodeImport(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2*nodeTransferLimit)
	var req nodeImportRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Content) > nodeTransferLimit {
		c.JSON(400, gin.H{"msg": "文件无效或超过 5 MB"})
		return
	}
	content := strings.TrimPrefix(req.Content, "\ufeff")
	var entries []models.NodeBackup
	switch req.Format {
	case "json":
		var backup nodeBackupFile
		if err := strictJSON(content, &backup); err != nil || backup.Format != "sublinkx-nodes" || backup.Version != 1 || backup.Nodes == nil {
			c.JSON(400, gin.H{"msg": "不是受支持的 sublinkX 节点备份文件（版本 1）"})
			return
		}
		entries = backup.Nodes
	case "txt":
		for _, line := range strings.Split(content, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				entries = append(entries, models.NodeBackup{Link: line, SourceType: "auto", Groups: []string{}})
				if len(entries) > nodeTransferCount {
					c.JSON(400, gin.H{"msg": "每次请导入 1 至 5000 条节点记录"})
					return
				}
			}
		}
	default:
		c.JSON(400, gin.H{"msg": "仅支持 JSON 和 TXT 文件"})
		return
	}
	if len(entries) == 0 || len(entries) > nodeTransferCount {
		c.JSON(400, gin.H{"msg": "每次请导入 1 至 5000 条节点记录"})
		return
	}
	existing, err := models.GetNodeList()
	if err != nil {
		c.JSON(500, gin.H{"msg": "读取节点失败"})
		return
	}
	seen := map[string]bool{}
	for _, n := range existing {
		kind := n.SourceType
		if kind == "" {
			kind = "auto"
		}
		seen[n.Link+"\x00"+kind] = true
	}
	rows := []nodeImportRow{}
	valid, skipped, invalid := 0, 0, 0
	for i := range entries {
		n := &entries[i]
		n.Link = strings.TrimSpace(n.Link)
		n.Name = strings.TrimSpace(n.Name)
		if n.SourceType == "" {
			n.SourceType = "auto"
		}
		reason := ""
		if len(n.Name) > 1024 || len(n.Link) > 65536 || len(n.Groups) > 100 {
			reason = "名称、链接或分组超出限制"
		}
		if n.SourceType != "auto" && n.SourceType != "proxy" && n.SourceType != "subscription" {
			reason = "链接类型无效"
		}
		if strings.ContainsRune(n.Link, 0) {
			reason = "链接包含无效字符"
		}
		groups := []string{}
		groupSeen := map[string]bool{}
		for _, group := range n.Groups {
			group = strings.TrimSpace(group)
			if len(group) > 256 || strings.ContainsAny(group, ",\x00\r\n") {
				reason = "分组名称无效"
			}
			if group != "" && !groupSeen[group] {
				groups = append(groups, group)
				groupSeen[group] = true
			}
		}
		n.Groups = groups
		if reason == "" {
			decoded, err := DocodeNodeName(&models.Node{Name: n.Name, Link: n.Link, SourceType: n.SourceType})
			if err != nil {
				reason = "节点链接无效或协议暂不支持"
			} else {
				n.Name = decoded.Name
			}
		}
		row := nodeImportRow{Index: i + 1, Name: n.Name, Status: "ready"}
		key := n.Link + "\x00" + n.SourceType
		if reason != "" {
			row.Status, row.Reason = "invalid", reason
			invalid++
		} else if seen[key] {
			row.Status, row.Reason = "duplicate", "链接和类型相同，跳过并保留原记录"
			skipped++
		} else {
			valid++
			seen[key] = true
		}
		rows = append(rows, row)
	}
	if !req.Confirm {
		c.JSON(200, gin.H{"code": "00000", "data": gin.H{"rows": rows, "valid": valid, "skipped": skipped, "invalid": invalid}, "msg": "预览完成，尚未保存"})
		return
	}
	if invalid > 0 {
		c.JSON(400, gin.H{"msg": "文件包含无效记录，请修正后重新导入；没有保存任何节点"})
		return
	}
	added, duplicates, err := models.ImportNodes(entries)
	if err != nil {
		c.JSON(500, gin.H{"msg": "导入失败，已回滚本次修改"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"added": added, "skipped": duplicates}, "msg": "导入完成"})
}
