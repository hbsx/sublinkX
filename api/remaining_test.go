package api

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sublink/middlewares"
	"sublink/models"
	"sublink/node"
	"sublink/utils"
	"testing"
)

const testProxy = "ss://YWVzLTEyOC1nY206cGFzcw@example.test:443#Proxy"
const testClash = "proxies: []\nproxy-groups:\n- name: All\n  type: select\n  proxies: [DIRECT]\n"

func postFixture(handler gin.HandlerFunc, values url.Values) *httptest.ResponseRecorder {
	r := gin.New()
	r.POST("/fixture", handler)
	w := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/fixture", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, request)
	return w
}
func mustTest(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func fixtureNode(t *testing.T, name string) models.Node {
	t.Helper()
	n := models.Node{Name: name, Link: testProxy}
	mustTest(t, n.Add())
	return n
}
func getFixture(sub models.Subcription, client string) *httptest.ResponseRecorder {
	r := gin.New()
	r.Use(middlewares.SafeRecovery(nil))
	r.GET("/c/", GetClient)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c/?token="+sub.Token+"&client="+client, nil))
	return w
}
func TestBadDeleteIDsCannotDeleteAnyNode(t *testing.T) {
	testDatabase(t)
	first := fixtureNode(t, "First")
	second := fixtureNode(t, "Second")
	r := gin.New()
	r.DELETE("/delete", NodeDel)
	for _, id := range []string{"bad", "0", "-1", "999999"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/delete?id="+id, nil))
		if w.Code < 400 {
			t.Fatalf("bad delete returned success: %s", id)
		}
		var count int64
		mustTest(t, models.DB.Model(&models.Node{}).Count(&count).Error)
		if count != 2 {
			t.Fatal("invalid ID deleted a node")
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/delete?id="+strconv.Itoa(second.ID), nil))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var remaining []models.Node
	mustTest(t, models.DB.Find(&remaining).Error)
	if len(remaining) != 1 || remaining[0].ID != first.ID {
		t.Fatal("valid ID deleted wrong node")
	}
}
func TestSubscriptionIDsSurviveRenameDuplicateAndCommaNames(t *testing.T) {
	testDatabase(t)
	a := fixtureNode(t, "Same, Name")
	b := fixtureNode(t, "Same, Name")
	// Node.Add de-duplicates identical records; use a different link for a distinct node.
	b = models.Node{Name: a.Name, Link: strings.Replace(testProxy, "example.test", "other.test", 1)}
	mustTest(t, b.Add())
	ids, _ := json.Marshal([]int{b.ID, a.ID})
	w := postFixture(SubAdd, url.Values{"name": {"DuplicateSub"}, "node_ids": {string(ids)}, "config": {"{}"}})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var subs []models.Subcription
	mustTest(t, models.DB.Find(&subs).Error)
	sub := subs[0]
	w = postFixture(NodeUpdadte, url.Values{"id": {strconv.Itoa(a.ID)}, "name": {"Renamed"}, "link": {testProxy}})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	stored := models.Subcription{}
	mustTest(t, stored.FindByToken(sub.Token))
	if len(stored.Nodes) != 2 || stored.Nodes[0].ID != b.ID || stored.Nodes[1].ID != a.ID {
		t.Fatal("node rename changed subscription membership/order")
	}
	other := models.Subcription{Name: sub.Name, Config: "{}", Nodes: []models.Node{a}}
	mustTest(t, other.Add())
	w = postFixture(SubUpdate, url.Values{"id": {strconv.Itoa(other.ID)}, "name": {"SecondRenamed"}, "node_ids": {string(ids)}, "config": {"{}"}})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	first, second := models.Subcription{}, models.Subcription{}
	mustTest(t, first.FindByToken(sub.Token))
	mustTest(t, second.FindByToken(other.Token))
	if first.Name != sub.Name || second.Name != "SecondRenamed" {
		t.Fatal("updated wrong same-name subscription")
	}
	if second.Token != other.Token {
		t.Fatal("edit rotated subscription token")
	}
}
func TestLegacyOrderingMigrationKeepsNodesAndTokens(t *testing.T) {
	testDatabase(t)
	a := fixtureNode(t, "A")
	b := fixtureNode(t, "B")
	sub := models.Subcription{Name: "Legacy", Config: "{}", Nodes: []models.Node{a, b}}
	mustTest(t, sub.Add())
	mustTest(t, models.DB.Model(&sub).Update("node_order", "B,A").Error)
	mustTest(t, models.MigrateSubscriptionOrder(models.DB))
	stored := models.Subcription{}
	mustTest(t, stored.FindByToken(sub.Token))
	if len(stored.Nodes) != 2 || stored.Nodes[0].ID != b.ID || stored.Token != sub.Token {
		t.Fatal("legacy migration lost order or credentials")
	}
	mustTest(t, models.DB.Model(&sub).Update("node_order", "OldName,B").Error)
	mustTest(t, models.MigrateSubscriptionOrder(models.DB))
	mustTest(t, stored.FindByToken(sub.Token))
	if len(stored.Nodes) != 2 {
		t.Fatal("migration lost node with stale legacy name")
	}
}
func TestSubscriptionUpdateRollsBackOnAssociationFailure(t *testing.T) {
	testDatabase(t)
	n := fixtureNode(t, "Node")
	sub := models.Subcription{Name: "Before", Config: "{}", Nodes: []models.Node{n}}
	mustTest(t, sub.Add())
	mustTest(t, models.DB.Exec("CREATE TRIGGER reject_association BEFORE INSERT ON subcription_nodes BEGIN SELECT RAISE(ABORT,'fixture failure'); END").Error)
	if err := sub.Update(&models.Subcription{Name: "After", Config: "{}", Nodes: []models.Node{n}}); err == nil {
		t.Fatal("expected association failure")
	}
	stored := models.Subcription{}
	mustTest(t, stored.FindByToken(sub.Token))
	if stored.Name != "Before" || len(stored.Nodes) != 1 {
		t.Fatal("failed update partially changed data")
	}
}
func TestLocalAndRemoteInvalidContentIsRejected(t *testing.T) {
	testDatabase(t)
	for _, link := range []string{"not-a-url", "trojan://pass@example.test:70000", "ss://bad"} {
		w := postFixture(NodeAdd, url.Values{"name": {"Named"}, "link": {link}, "source_type": {"proxy"}})
		if w.Code < 400 {
			t.Fatal("named invalid node accepted")
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("<html>Login required</html>")) }))
	defer server.Close()
	n := models.Node{Name: "Remote", Link: server.URL, SourceType: "subscription"}
	mustTest(t, n.Add())
	sub := models.Subcription{Name: "RemoteSub", Config: "{}", Nodes: []models.Node{n}}
	mustTest(t, sub.Add())
	for _, client := range []string{"v2ray", "clash", "surge"} {
		if w := getFixture(sub, client); w.Code != 502 {
			t.Fatalf("invalid remote returned %d", w.Code)
		}
	}
}
func TestCommaALPNAndHTTPSManagedURL(t *testing.T) {
	testDatabase(t)
	n := models.Node{Name: "ALPN", Link: "anytls://test-password@example.test:443?alpn=h2,http/1.1#ALPN", SourceType: "proxy"}
	mustTest(t, n.Add())
	sub := models.Subcription{Name: "Comma", Config: "{}", Nodes: []models.Node{n}}
	mustTest(t, sub.Add())
	w := getFixture(sub, "v2ray")
	if w.Code != 200 || strings.TrimSpace(node.Base64Decode(w.Body.String())) != n.Link {
		t.Fatal("comma split corrupted URL")
	}
	file := filepath.Join(t.TempDir(), "surge.conf")
	mustTest(t, os.WriteFile(file, []byte("[Proxy]\nDIRECT = direct\n[Proxy Group]\nAll = select, DIRECT\n[Rule]\nFINAL,All\n"), 0600))
	config, _ := json.Marshal(node.SqlConfig{Surge: file})
	sub.Config = string(config)
	sub.Nodes = []models.Node{{Link: testProxy}}
	w = httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "http://public.example/c/?token=dummy&client=surge", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Set("subscription", &sub)
	GetSurge(c)
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "#!MANAGED-CONFIG https://public.example/") {
		t.Fatal("reverse proxy HTTPS not preserved")
	}
}
func TestUsedTemplateCannotBeRenamedOrDeleted(t *testing.T) {
	testDatabase(t)
	mustTest(t, InitTemplateDir())
	file := "./template/custom.yaml"
	mustTest(t, os.WriteFile(file, []byte(testClash), 0600))
	n := fixtureNode(t, "TemplateNode")
	config, _ := json.Marshal(node.SqlConfig{Clash: file})
	sub := models.Subcription{Name: "Template", Config: string(config), Nodes: []models.Node{n}}
	mustTest(t, sub.Add())
	if getFixture(sub, "clash").Code != 200 {
		t.Fatal("initial subscription failed")
	}
	if w := postFixture(UpdateTemp, url.Values{"oldname": {"custom.yaml"}, "filename": {"renamed.yaml"}, "text": {testClash}}); w.Code != 409 {
		t.Fatal("used template rename was allowed")
	}
	if w := postFixture(DelTemp, url.Values{"filename": {"custom.yaml"}}); w.Code != 409 {
		t.Fatal("used template deletion was allowed")
	}
	if w := postFixture(UpdateTemp, url.Values{"oldname": {"custom.yaml"}, "filename": {"custom.yaml"}, "text": {testClash}}); w.Code != 200 {
		t.Fatal("in-place template update failed")
	}
	if getFixture(sub, "clash").Code != 200 {
		t.Fatal("blocked mutation broke subscription")
	}
	if w := postFixture(AddTemp, url.Values{"filename": {"unused.yaml"}, "text": {testClash}}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := postFixture(UpdateTemp, url.Values{"oldname": {"unused.yaml"}, "filename": {"unused2.yaml"}, "text": {testClash}}); w.Code != 200 {
		t.Fatal("unused template rename failed")
	}
}
func TestSafeRecoveryNeverLogsCredentials(t *testing.T) {
	for _, mode := range []string{gin.DebugMode, gin.ReleaseMode} {
		gin.SetMode(mode)
		output := new(bytes.Buffer)
		r := gin.New()
		r.Use(middlewares.SafeRecovery(output))
		r.GET("/fixture", func(c *gin.Context) { panic("secret-panic-value") })
		request := httptest.NewRequest("GET", "/fixture?token=secret-subscription", nil)
		request.Header.Set("Authorization", "Bearer secret-login")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)
		if w.Code != 500 || strings.Contains(output.String(), "secret-") {
			t.Fatal("recovery leaked credentials")
		}
	}
	gin.SetMode(gin.TestMode)
}

type fixtureTransport func(*http.Request) (*http.Response, error)

func (fn fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }
func TestAccessLogsUseSubscriptionID(t *testing.T) {
	testDatabase(t)
	n := fixtureNode(t, "Node")
	a := models.Subcription{Name: "SameSub", Config: "{}", Nodes: []models.Node{n}}
	b := a
	mustTest(t, a.Add())
	mustTest(t, b.Add())
	previous := utils.HTTPClient
	utils.HTTPClient = &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"addr":"Test","ip":"192.0.2.1"}`))}, nil
	})}
	defer func() { utils.HTTPClient = previous }()
	r := gin.New()
	r.Use(middlewares.GetIp)
	r.GET("/c/", GetClient)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c/?token="+b.Token, nil))
	var logs []models.SubLogs
	mustTest(t, models.DB.Find(&logs).Error)
	if w.Code != 200 || len(logs) != 1 || logs[0].SubcriptionID != b.ID {
		t.Fatal("visit logged to wrong same-name subscription")
	}
}

func TestNodeGroupsUseIDsAndRollbackTogether(t *testing.T) {
	testDatabase(t)
	for _, host := range []string{"example.test", "other.test"} {
		w := postFixture(NodeAdd, url.Values{"name": {"Same"}, "link": {strings.Replace(testProxy, "example.test", host, 1)}, "group": {"One, Two"}})
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	nodes, err := models.GetNodeList()
	mustTest(t, err)
	if len(nodes) != 2 || len(nodes[0].GroupNodes) != 2 || len(nodes[1].GroupNodes) != 2 {
		t.Fatal("same-name nodes were linked to the wrong groups")
	}
	w := postFixture(GroupNodeSet, url.Values{"id": {strconv.Itoa(nodes[1].ID)}, "group": {"Third"}})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	nodes, err = models.GetNodeList()
	mustTest(t, err)
	if len(nodes[0].GroupNodes) != 2 || len(nodes[1].GroupNodes) != 1 || nodes[1].GroupNodes[0].Name != "Third" {
		t.Fatal("group edit changed another same-name node")
	}
	mustTest(t, models.DB.Exec("CREATE TRIGGER reject_group BEFORE INSERT ON group_node_nodes BEGIN SELECT RAISE(ABORT,'fixture failure'); END").Error)
	w = postFixture(NodeUpdadte, url.Values{"id": {strconv.Itoa(nodes[1].ID)}, "name": {"After"}, "link": {testProxy}, "group": {"NewGroup"}})
	if w.Code < 400 {
		t.Fatal("association failure was accepted")
	}
	stored := models.Node{}
	mustTest(t, models.DB.Preload("GroupNodes").First(&stored, nodes[1].ID).Error)
	if stored.Name != "Same" || len(stored.GroupNodes) != 1 || stored.GroupNodes[0].Name != "Third" {
		t.Fatal("failed group edit partially changed data")
	}
	w = postFixture(NodeAdd, url.Values{"name": {"Rejected"}, "link": {testProxy}, "group": {"NewGroup"}})
	if w.Code < 400 {
		t.Fatal("failed add was accepted")
	}
	var count int64
	mustTest(t, models.DB.Model(&models.Node{}).Count(&count).Error)
	if count != 2 {
		t.Fatal("failed add left a partially created node")
	}
}

func TestDeleteNodeClearsAssociationsAndOnlyItsEmptyGroups(t *testing.T) {
	testDatabase(t)
	a, b := fixtureNode(t, "A"), fixtureNode(t, "B")
	mustTest(t, a.UpdateGroup(groupForm("Shared,OnlyA")))
	mustTest(t, b.UpdateGroup(groupForm("Shared")))
	sub := models.Subcription{Name: "DeleteFixture", Config: "{}", Nodes: []models.Node{a, b}}
	mustTest(t, sub.Add())
	mustTest(t, a.Del())
	mustTest(t, sub.FindByToken(sub.Token))
	groups, err := models.GetGroupNodeList()
	mustTest(t, err)
	if len(sub.Nodes) != 1 || sub.Nodes[0].ID != b.ID || len(groups) != 1 || groups[0].Name != "Shared" || len(groups[0].Nodes) != 1 {
		t.Fatal("node deletion lost unrelated data or left stale associations")
	}
}
