package models

import "gorm.io/gorm"

// NodeBackup deliberately excludes database IDs, timestamps and subscriptions.
type NodeBackup struct {
	Name       string   `json:"name"`
	Link       string   `json:"link"`
	SourceType string   `json:"source_type"`
	Groups     []string `json:"groups"`
}

func sourceKind(kind string) string {
	if kind == "" {
		return "auto"
	}
	return kind
}

// ImportNodes rechecks duplicates inside the transaction, including repeated imports.
func ImportNodes(entries []NodeBackup) (added, skipped int, err error) {
	err = DB.Transaction(func(tx *gorm.DB) error {
		var existing []Node
		if err := tx.Find(&existing).Error; err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, n := range existing {
			seen[n.Link+"\x00"+sourceKind(n.SourceType)] = true
		}
		for _, entry := range entries {
			key := entry.Link + "\x00" + sourceKind(entry.SourceType)
			if seen[key] {
				skipped++
				continue
			}
			n := Node{Name: entry.Name, Link: entry.Link, SourceType: entry.SourceType}
			if err := tx.Omit("GroupNodes").Create(&n).Error; err != nil {
				return err
			}
			groups := []GroupNode{}
			for _, name := range entry.Groups {
				groups = append(groups, GroupNode{Name: name})
			}
			if err := replaceNodeGroups(tx, &n, groups); err != nil {
				return err
			}
			seen[key] = true
			added++
		}
		return nil
	})
	if err != nil {
		added, skipped = 0, 0
	}
	return
}
