package models

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type Subcription struct {
	gorm.Model
	ID        int
	Name      string
	Token     string    `gorm:"uniqueIndex"`
	Config    string    `gorm:"type:text"`
	NodeOrder string    `gorm:"type:text"`
	Nodes     []Node    `gorm:"many2many:subcription_nodes;"`
	SubLogs   []SubLogs `gorm:"foreignKey:SubcriptionID;"`
}
type SubscriptionConfig struct {
	Clash string `json:"clash"`
	Surge string `json:"surge"`
	UDP   bool   `json:"udp"`
	Cert  bool   `json:"cert"`
}

func nodeOrder(nodes []Node) string {
	ids := []int{}
	seen := map[int]bool{}
	for _, n := range nodes {
		if !seen[n.ID] {
			ids = append(ids, n.ID)
			seen[n.ID] = true
		}
	}
	data, _ := json.Marshal(ids)
	return string(data)
}

// Fall back to all remaining associations when migrating stale legacy names.
func (sub *Subcription) sortNodes() {
	var ids []int
	byID := map[int]Node{}
	for _, n := range sub.Nodes {
		byID[n.ID] = n
	}
	ordered := []Node{}
	seen := map[int]bool{}
	appendNode := func(n Node) {
		if !seen[n.ID] {
			ordered = append(ordered, n)
			seen[n.ID] = true
		}
	}
	if json.Unmarshal([]byte(sub.NodeOrder), &ids) == nil {
		for _, id := range ids {
			if n, ok := byID[id]; ok {
				appendNode(n)
			}
		}
	} else {
		for _, name := range strings.Split(sub.NodeOrder, ",") {
			for _, n := range sub.Nodes {
				if n.Name == strings.TrimSpace(name) {
					appendNode(n)
				}
			}
		}
	}
	for _, n := range sub.Nodes {
		appendNode(n)
	}
	sub.Nodes = ordered
}
func subscriptionQuery(tx *gorm.DB, id int, name string) (*gorm.DB, error) {
	if id > 0 {
		return tx.Where("id = ?", id), nil
	}
	if name == "" {
		return nil, fmt.Errorf("订阅 ID 无效")
	}
	var count int64
	if err := tx.Model(&Subcription{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, fmt.Errorf("订阅不存在或同名，请按 ID 操作")
	}
	return tx.Where("name = ?", name), nil
}
func (sub *Subcription) Add() error {
	var err error
	sub.Token, err = RandomToken()
	if err != nil {
		return err
	}
	sub.NodeOrder = nodeOrder(sub.Nodes)
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Nodes", "SubLogs").Create(sub).Error; err != nil {
			return err
		}
		return tx.Model(sub).Omit("Nodes.*").Association("Nodes").Replace(sub.Nodes)
	})
}
func (sub *Subcription) Update(next *Subcription) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		q, err := subscriptionQuery(tx, sub.ID, sub.Name)
		if err != nil {
			return err
		}
		var existing Subcription
		if err = q.First(&existing).Error; err != nil {
			return err
		}
		if err = tx.Model(&existing).Updates(map[string]interface{}{"name": next.Name, "config": next.Config, "node_order": nodeOrder(next.Nodes)}).Error; err != nil {
			return err
		}
		return tx.Model(&existing).Omit("Nodes.*").Association("Nodes").Replace(next.Nodes)
	})
}
func (sub *Subcription) Find() error {
	q, err := subscriptionQuery(DB, sub.ID, sub.Name)
	if err != nil {
		return err
	}
	if err = q.Preload("Nodes").Preload("SubLogs").First(sub).Error; err != nil {
		return err
	}
	sub.sortNodes()
	return nil
}
func (sub *Subcription) FindByToken(token string) error {
	if err := DB.Preload("Nodes").Where("token = ?", token).First(sub).Error; err != nil {
		return err
	}
	sub.sortNodes()
	return nil
}
func (sub *Subcription) List() ([]Subcription, error) {
	subs := []Subcription{}
	if err := DB.Preload("Nodes").Preload("SubLogs").Find(&subs).Error; err != nil {
		return nil, err
	}
	for i := range subs {
		subs[i].sortNodes()
	}
	return subs, nil
}
func (sub *Subcription) IPlogUpdate() error {
	return DB.Model(sub).Association("SubLogs").Replace(sub.SubLogs)
}
func (sub *Subcription) Del() error {
	if sub.ID <= 0 {
		return fmt.Errorf("订阅 ID 无效")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&Subcription{}, sub.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(sub).Association("Nodes").Clear(); err != nil {
			return err
		}
		if err := tx.Where("subcription_id = ?", sub.ID).Delete(&SubLogs{}).Error; err != nil {
			return err
		}
		return tx.Delete(sub).Error
	})
}
func MigrateSubscriptionOrder(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var subs []Subcription
		if err := tx.Preload("Nodes").Find(&subs).Error; err != nil {
			return err
		}
		for _, sub := range subs {
			sub.sortNodes()
			order := nodeOrder(sub.Nodes)
			if order != sub.NodeOrder {
				if err := tx.Model(&Subcription{}).Where("id = ?", sub.ID).Update("node_order", order).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
