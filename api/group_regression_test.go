package api

import (
	"sublink/models"
	"testing"
)

func TestGroupRenamePreservesIdentityAndRejectsDuplicates(t *testing.T) {
	testDatabase(t)
	n := models.Node{Name: "Grouped", Link: "ss://YWVzLTEyOC1nY206cGFzcw@example.test:443"}
	if err := n.AddWithGroups([]models.GroupNode{{Name: "Original"}}); err != nil {
		t.Fatal(err)
	}
	var group models.GroupNode
	if err := models.DB.Where("name = ?", "Original").First(&group).Error; err != nil {
		t.Fatal(err)
	}
	id := group.ID
	if err := group.Update(&models.GroupNode{Name: "Renamed"}); err != nil {
		t.Fatal("valid group rename failed", err)
	}
	var stored models.GroupNode
	if err := models.DB.Preload("Nodes").First(&stored, id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Name != "Renamed" || len(stored.Nodes) != 1 || stored.Nodes[0].ID != n.ID {
		t.Fatal("group rename broke associations")
	}
	other := models.GroupNode{Name: "Taken"}
	if err := other.Add(); err != nil {
		t.Fatal(err)
	}
	if err := group.Update(&models.GroupNode{Name: "Taken"}); err == nil {
		t.Fatal("duplicate group rename succeeded")
	}
	if err := models.DB.First(&stored, id).Error; err != nil || stored.Name != "Renamed" {
		t.Fatal("failed rename changed original")
	}
	reused := models.GroupNode{Name: "Renamed"}
	if err := reused.Add(); err != nil || reused.ID != id {
		t.Fatal("reused group lost its identity")
	}
	if err := (&models.GroupNode{Name: "Missing"}).Ass(&n); err == nil {
		t.Fatal("missing group association did not fail")
	}
}
