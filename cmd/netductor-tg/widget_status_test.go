package main

import (
	"os"
	"strings"
	"testing"
)

func TestStatusWidgetHTMLUnder32KiB(t *testing.T) {
	if len(statusWidgetHTML()) > 32768 {
		t.Fatalf("statusWidgetHTML %d > 32768", len(statusWidgetHTML()))
	}
}

func TestChannelWidgetHTMLUnder32KiB(t *testing.T) {
	if len(channelWidgetHTML()) > 32768 {
		t.Fatalf("channelWidgetHTML %d > 32768", len(channelWidgetHTML()))
	}
}

func TestStatusWidgetChatDataJSONUnder1KiB(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	s, err := statusWidgetChatDataJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) > 1024 {
		t.Fatalf("chat_data %d > 1024", len(s))
	}
	if !strings.Contains(s, `"ver"`) {
		t.Fatalf("missing ver: %s", s)
	}
}

func TestChannelWidgetChatDataJSONUnder1KiB(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	s, err := channelWidgetChatDataJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) > 1024 {
		t.Fatalf("chat_data %d > 1024", len(s))
	}
}

func TestWidgetsEnabledGate(t *testing.T) {
	os.Unsetenv("NETDUCTOR_TG_WIDGETS")
	if widgetsEnabled() {
		t.Fatal("expected disabled")
	}
	t.Setenv("NETDUCTOR_TG_WIDGETS", "1")
	if !widgetsEnabled() {
		t.Fatal("expected enabled")
	}
}

func TestParseWidgetAct(t *testing.T) {
	sc, act := parseWidgetAct("status:refresh")
	if sc != "status" || act != "refresh" {
		t.Fatalf("got %q %q", sc, act)
	}
	sc, act = parseWidgetAct("refresh")
	if sc != "app" || act != "refresh" {
		t.Fatalf("spa short act got %q %q", sc, act)
	}
	sc, act = parseWidgetAct("g:status")
	if sc != "app" || act != "g:status" {
		t.Fatalf("g: got %q %q", sc, act)
	}
}

func TestWidgetRegistryHasStatusAndChannel(t *testing.T) {
	if widgetRegistry["status"] == nil {
		t.Fatal("status not registered")
	}
	if widgetRegistry["channel"] == nil {
		t.Fatal("channel not registered")
	}
}

func TestPhase1WidgetsRegistered(t *testing.T) {
	for _, id := range []string{"netmap", "secondary", "probes", "versions"} {
		if widgetRegistry[id] == nil {
			t.Fatalf("missing %s", id)
		}
		if len(widgetRegistry[id].HTML) > 32768 {
			t.Fatalf("%s HTML too large", id)
		}
	}
}

func TestPhase1ChatData(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	for _, fn := range []struct {
		name string
		f    func() (string, error)
		max  int
	}{
		{"netmap", netmapWidgetChatDataJSON, 1500},
		{"secondary", secondaryWidgetChatDataJSON, 1024},
		{"versions", versionsWidgetChatDataJSON, 1024},
	} {
		s, err := fn.f()
		if err != nil {
			t.Fatalf("%s: %v", fn.name, err)
		}
		if len(s) > fn.max {
			t.Fatalf("%s len %d > %d", fn.name, len(s), fn.max)
		}
	}
}

func TestPhase2ListWidgetsRegistered(t *testing.T) {
	for _, id := range []string{"users", "devices", "nodes", "sites", "pending", "sshhosts", "backup", "seclist", "fleet"} {
		if widgetRegistry[id] == nil {
			t.Fatalf("missing list widget %s", id)
		}
		if len(widgetRegistry[id].HTML) > 32768 {
			t.Fatalf("%s HTML too large", id)
		}
	}
}

func TestPhase3CardWidgetsRegistered(t *testing.T) {
	for _, id := range []string{"user", "node", "operator", "addons", "svcpaths"} {
		if widgetRegistry[id] == nil {
			t.Fatalf("missing card widget %s", id)
		}
		if len(widgetRegistry[id].HTML) > 32768 {
			t.Fatalf("%s HTML too large", id)
		}
	}
}

func TestPhase4ResidualWidgetsRegistered(t *testing.T) {
	for _, id := range []string{"digest", "actions", "audit", "sessions", "routers", "lampac", "canary"} {
		if widgetRegistry[id] == nil {
			t.Fatalf("missing phase4 widget %s", id)
		}
	}
}

func TestAppWidgetRegistered(t *testing.T) {
	if widgetRegistry["app"] == nil {
		t.Fatal("app not registered")
	}
	if len(widgetRegistry["app"].HTML) > 32768 {
		t.Fatal("app HTML too large")
	}
}
