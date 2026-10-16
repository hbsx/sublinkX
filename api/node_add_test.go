package api

import (
	"net/url"
	"sublink/models"
	"testing"
)

func TestNodeAddDistinguishesSourceTypesWithoutChangingOtherManualDuplicates(t *testing.T) {
	testDatabase(t)
	link := "http://user:pass@example.test:443"
	add := func(name, target, kind, group string) {
		t.Helper()
		result := postFixture(NodeAdd, url.Values{
			"name":        {name},
			"link":        {target},
			"source_type": {kind},
			"group":       {group},
		})
		if result.Code != 200 {
			t.Fatalf("add %q/%q returned %d: %s", name, kind, result.Code, result.Body.String())
		}
	}
	find := func(name, target, kind string) models.Node {
		t.Helper()
		nodes, err := models.GetNodeList()
		mustTest(t, err)
		for _, n := range nodes {
			if n.Name == name && n.Link == target && n.SourceType == kind {
				return n
			}
		}
		t.Fatalf("missing node %q/%q/%q among %d nodes", name, target, kind, len(nodes))
		return models.Node{}
	}
	count := func(want int) {
		t.Helper()
		var got int64
		mustTest(t, models.DB.Model(&models.Node{}).Count(&got).Error)
		if got != int64(want) {
			t.Fatalf("got %d nodes; want %d", got, want)
		}
	}

	add("Same", link, "proxy", "ProxyGroup")
	proxy := find("Same", link, "proxy")
	add("Same", link, "subscription", "SubscriptionGroup")
	subscription := find("Same", link, "subscription")
	count(2)
	if proxy.ID == subscription.ID {
		t.Fatal("different source types reused the same node")
	}

	add("Same", link, "proxy", "ExtraGroup")
	count(2)
	proxy = find("Same", link, "proxy")
	subscription = find("Same", link, "subscription")
	groups := map[string]bool{}
	for _, group := range proxy.GroupNodes {
		groups[group.Name] = true
	}
	if len(groups) != 2 || !groups["ProxyGroup"] || !groups["ExtraGroup"] || len(subscription.GroupNodes) != 1 || subscription.GroupNodes[0].Name != "SubscriptionGroup" {
		t.Fatalf("repeated add changed unrelated groups: proxy=%v subscription=%v", proxy.GroupNodes, subscription.GroupNodes)
	}

	add("Alias", link, "proxy", "")
	add("Same", "http://user:pass@other.test:443", "proxy", "")
	count(4)
	find("Alias", link, "proxy")
	find("Same", "http://user:pass@other.test:443", "proxy")

	legacy := models.Node{Name: "Legacy", Link: testProxy}
	mustTest(t, legacy.Add())
	add("Legacy", testProxy, "auto", "")
	count(5)
	var stored models.Node
	mustTest(t, models.DB.First(&stored, legacy.ID).Error)
	if stored.SourceType != "" {
		t.Fatal("adding auto duplicated or changed a legacy empty source type")
	}
}
