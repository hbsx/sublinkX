package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http/httptest"
	"sublink/models"
	"testing"
)

func transferRequest(t *testing.T, format, content string, confirm bool) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(nodeImportRequest{Format: format, Content: content, Confirm: confirm})
	mustTest(t, err)
	r := gin.New()
	r.POST("/import", NodeImport)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}
func backupContent(t *testing.T, entries []models.NodeBackup) string {
	t.Helper()
	body, err := json.Marshal(nodeBackupFile{Format: "sublinkx-nodes", Version: 1, Nodes: entries})
	mustTest(t, err)
	return string(body)
}
func TestNodeTransferPreviewRoundTripAndRepeatedImport(t *testing.T) {
	testDatabase(t)
	old := fixtureNode(t, "Existing")
	content := backupContent(t, []models.NodeBackup{
		{Name: "Custom name", Link: testProxy + "-new", SourceType: "proxy", Groups: []string{"A", "B", "A"}},
		{Name: "Renamed duplicate", Link: old.Link, SourceType: "auto", Groups: []string{"Do not add"}},
		{Name: "File duplicate", Link: testProxy + "-new", SourceType: "proxy", Groups: []string{"Do not add"}},
	})
	preview := transferRequest(t, "json", content, false)
	if preview.Code != 200 {
		t.Fatal(preview.Body.String())
	}
	var response struct {
		Data struct{ Valid, Skipped, Invalid int }
	}
	mustTest(t, json.Unmarshal(preview.Body.Bytes(), &response))
	if response.Data.Valid != 1 || response.Data.Skipped != 2 || response.Data.Invalid != 0 {
		t.Fatal(preview.Body.String())
	}
	before, err := models.GetNodeList()
	mustTest(t, err)
	if len(before) != 1 {
		t.Fatal("preview wrote to database")
	}
	for i := 0; i < 2; i++ {
		result := transferRequest(t, "json", content, true)
		if result.Code != 200 {
			t.Fatal(result.Body.String())
		}
	}
	nodes, err := models.GetNodeList()
	mustTest(t, err)
	if len(nodes) != 2 {
		t.Fatalf("duplicate import created %d nodes", len(nodes))
	}
	for _, n := range nodes {
		if n.ID == old.ID {
			if n.Name != "Existing" || len(n.GroupNodes) != 0 {
				t.Fatal("existing node changed")
			}
			continue
		}
		if n.Name != "Custom name" || n.SourceType != "proxy" || len(n.GroupNodes) != 2 {
			t.Fatalf("metadata lost: %+v", n)
		}
	}
}
func TestNodeTransferInvalidFileSavesNothing(t *testing.T) {
	testDatabase(t)
	cases := []struct{ format, content string }{
		{"txt", testProxy + "\nnot-a-node"},
		{"json", `{"format":"sublinkx-nodes","version":2,"nodes":[]}`},
		{"json", `{"format":"sublinkx-nodes","version":1,"nodes":[{"link":"x","id":7}]}`},
		{"txt", ""},
	}
	for _, tc := range cases {
		if result := transferRequest(t, tc.format, tc.content, true); result.Code != 400 {
			t.Fatal(result.Body.String())
		}
	}
	nodes, err := models.GetNodeList()
	mustTest(t, err)
	if len(nodes) != 0 {
		t.Fatal("invalid file partially saved")
	}
}
func TestNodeTransferTXTAndTransactionRollback(t *testing.T) {
	testDatabase(t)
	body := "\ufeff" + testProxy + "\r\n" + testProxy + "\n"
	result := transferRequest(t, "txt", body, true)
	if result.Code != 200 {
		t.Fatal(result.Body.String())
	}
	nodes, err := models.GetNodeList()
	mustTest(t, err)
	if len(nodes) != 1 || nodes[0].Name != "Proxy" {
		t.Fatal("TXT BOM, names or duplicate handling failed")
	}
	mustTest(t, models.DB.Callback().Create().Before("gorm:create").Register("fail-import-group", func(tx *gorm.DB) {
		if tx.Statement.Table == "group_nodes" {
			tx.AddError(errors.New("forced group write failure"))
		}
	}))
	defer models.DB.Callback().Create().Remove("fail-import-group")
	content := backupContent(t, []models.NodeBackup{
		{Name: "First", Link: testProxy + "-first", SourceType: "proxy"},
		{Name: "Second", Link: testProxy + "-second", SourceType: "proxy", Groups: []string{"fail"}},
	})
	if result := transferRequest(t, "json", content, true); result.Code != 500 {
		t.Fatal(result.Body.String())
	}
	nodes, err = models.GetNodeList()
	mustTest(t, err)
	if len(nodes) != 1 {
		t.Fatal("failed import did not roll back")
	}
}
