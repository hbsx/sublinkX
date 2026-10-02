package models

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

type GroupNode struct {
	gorm.Model
	ID    int
	Name  string
	Nodes []Node `gorm:"many2many:group_node_nodes"` // 多对多关联字段
}

type Node struct {
	gorm.Model
	ID         int
	Name       string
	Link       string
	SourceType string
	GroupNodes []GroupNode `gorm:"many2many:group_node_nodes"` // 反向关联字段
}

// hook Node 写入创建删除修改 等写入权限
func (n *Node) AfterSave(*gorm.DB) error {
	// 写操作前执行（Create 或 Update）

	return nil
}

// 创建分组
func (gn *GroupNode) Add() error {
	// 检查分组是否已存在
	var existingGroup GroupNode
	result := DB.Model(gn).Where("name = ?", gn.Name).First(&existingGroup) // 查询数据库中是否存在同名的分组
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Println(result.Error)
		return result.Error // 如果查询出错，返回错误
	}
	if result.RowsAffected > 0 { // 如果查询到分组已存在
		// log.Println("分组已存在")
		return nil // 不返回错误存在就跳过

	}
	return DB.FirstOrCreate(gn, GroupNode{Name: gn.Name}).Error
}

// 关联分组
func (gn *GroupNode) Ass(n *Node) error {
	result := DB.Model(gn).Where("name = ?", gn.Name).First(gn) // 查找分组
	// log.Println("分组ID:", gn.ID, "分组昵称:", gn.Name, "错误信息:", result.Error)
	if result.Error != nil {
		log.Println(result.Error)
	}
	result = DB.Model(n).Where("name = ?", n.Name).First(n) // 查找节点
	// log.Println("节点ID:", n.ID, "节点昵称:", n.Name, "错误信息:", result.Error)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return DB.Model(&gn).Association("Nodes").Append(n)
}

// 更新分组信息
func (gn *GroupNode) Update(NewGn *GroupNode) error {
	// 读取分组数据
	var FirstGn GroupNode
	result := DB.Model(gn).Where("id = ? or name = ?", NewGn.ID, NewGn.Name).First(&FirstGn)
	if result.Error != nil {
		log.Println(result.Error)
		return result.Error
	}
	if result.RowsAffected > 0 {
		return errors.New("分组已存在")
	}
	return DB.Model(gn).Where("id = ? or name = ?", gn.ID, gn.Name).Updates(&NewGn).Error
}

// 删除分组
func (gn *GroupNode) Del() error {
	if gn.ID <= 0 {
		return errors.New("分组 ID 无效")
	}
	// 读取分组数据
	result := DB.Model(gn).Where("id = ?", gn.ID).First(&gn)
	if result.Error != nil {
		log.Println(result.Error)
		return result.Error
	}
	// 读取分组关联的节点数据
	result = DB.Model(gn).Preload("Nodes").First(gn) // 预加载分组关联的节点数据
	if result.Error != nil {
		log.Println(result.Error)
		return result.Error
	}
	log.Println("解除分组节点关联")
	err := DB.Model(gn).Association("Nodes").Delete(gn.Nodes)
	if err != nil {
		log.Println("解除关联失败", err)
		return err
	}
	return DB.Model(gn).Where("id = ? or name = ?", gn.ID, gn.Name).Delete(gn).Error // 删除分组记录
}

// 查看所有分组

func GetGroupNodeList() ([]GroupNode, error) {
	var gns []GroupNode
	result := DB.Model(gns).Preload("Nodes").Find(&gns)
	if result.Error != nil {
		return nil, errors.New("没有任何分组")
	}
	return gns, result.Error
}

/* 下面为节点的增删改查 */

// 添加节点的方法
func (n *Node) Add() error {
	return addNode(DB, n)
}

func addNode(tx *gorm.DB, n *Node) error {
	// 检查节点是否已存在
	var existingNode Node
	result := tx.Where("link = ? and name =?", n.Link, n.Name).First(&existingNode)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error // 如果查询出错，返回错误
	}
	if result.RowsAffected > 0 {
		// log.Println("节点已经存在")
		*n = existingNode
		return nil // Reuse the persisted identity.
	}
	return tx.Omit("GroupNodes").Create(n).Error
}

func (n *Node) AddWithGroups(groups []GroupNode) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := addNode(tx, n); err != nil {
			return err
		}
		var stored Node
		if err := tx.Preload("GroupNodes").First(&stored, n.ID).Error; err != nil {
			return err
		}
		return replaceNodeGroups(tx, n, append(stored.GroupNodes, groups...))
	})
}

// 删除节点
func (n *Node) Del() error {
	if n.ID <= 0 {
		return errors.New("节点 ID 无效")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing Node
		if err := tx.Preload("GroupNodes").First(&existing, n.ID).Error; err != nil {
			return err
		}
		previousGroups := append([]GroupNode(nil), existing.GroupNodes...)
		if err := tx.Model(&existing).Association("GroupNodes").Clear(); err != nil {
			return err
		}
		if err := tx.Table("subcription_nodes").Where("node_id = ?", existing.ID).Delete(nil).Error; err != nil {
			return err
		}
		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}
		for _, group := range previousGroups {
			if tx.Model(&group).Association("Nodes").Count() == 0 {
				if err := tx.Delete(&group).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func (n *Node) UpdateNode(next *Node) error {
	if n.ID <= 0 {
		return errors.New("节点 ID 无效")
	}
	if err := DB.First(&Node{}, n.ID).Error; err != nil {
		return err
	}
	return DB.Model(&Node{}).Where("id = ?", n.ID).Updates(map[string]interface{}{"name": next.Name, "link": next.Link, "source_type": next.SourceType}).Error
}

// 检查分组无绑定则删除
func IsGroupNotDel(gns []GroupNode) error {
	// 如果分组节点没有关联的节点则删除分组节点
	for _, gn := range gns {
		DB.Model(gn).Preload("Nodes").Find(&gn) // 预加载分组节点数据
		// log.Println("gnNodes:", gn.Nodes, "长度:", len(gn.Nodes))
		if len(gn.Nodes) == 0 {
			// log.Println("分组节点没有关联的节点，删除分组节点", gn.Name)
			err := DB.Model(gn).Delete(&gn).Error // 删除分组节点
			if err != nil {
				log.Println("删除分组节点失败", err)
				return err
			}
		}
	}
	return nil
}

// 更新关联分组

func resolveNode(tx *gorm.DB, n *Node) error {
	if n.ID > 0 {
		return tx.First(n, n.ID).Error
	}
	if n.Name == "" {
		return errors.New("节点 ID 无效")
	}
	var matches []Node
	if err := tx.Where("name = ?", n.Name).Find(&matches).Error; err != nil {
		return err
	}
	if len(matches) != 1 {
		return errors.New("节点不存在或同名，请按 ID 选择")
	}
	*n = matches[0]
	return nil
}
func replaceNodeGroups(tx *gorm.DB, n *Node, groups []GroupNode) error {
	resolved := []GroupNode{}
	seen := map[string]bool{}
	for _, group := range groups {
		if group.Name == "" || seen[group.Name] {
			continue
		}
		seen[group.Name] = true
		var actual GroupNode
		if err := tx.Where("name = ?", group.Name).FirstOrCreate(&actual, GroupNode{Name: group.Name}).Error; err != nil {
			return err
		}
		resolved = append(resolved, actual)
	}
	return tx.Model(n).Omit("GroupNodes.*").Association("GroupNodes").Replace(resolved)
}
func (n *Node) UpdateGroup(groups []GroupNode) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := resolveNode(tx, n); err != nil {
			return err
		}
		return replaceNodeGroups(tx, n, groups)
	})
}
func (n *Node) UpdateNodeAndGroups(next *Node, groups []GroupNode) error {
	if n.ID <= 0 {
		return errors.New("节点 ID 无效")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := resolveNode(tx, n); err != nil {
			return err
		}
		if err := tx.Model(&Node{}).Where("id = ?", n.ID).Updates(map[string]interface{}{"name": next.Name, "link": next.Link, "source_type": next.SourceType}).Error; err != nil {
			return err
		}
		return replaceNodeGroups(tx, n, groups)
	})
}

// 查看所有节点

func GetNodeList() ([]Node, error) {
	var ns []Node
	result := DB.Model(ns).Preload("GroupNodes").Find(&ns)
	if result.Error != nil {
		return nil, result.Error
	}
	return ns, result.Error
}
