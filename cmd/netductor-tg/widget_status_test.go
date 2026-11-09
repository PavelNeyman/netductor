package main

import (
	"os"
	"strings"
	"testing"
)

func TestStatusWidgetHTMLUnder32KiB(t *testing.T) {
	if len(statusWidgetHTML) > 32768 {
		t.Fatalf("statusWidgetHTML %d > 32768", len(statusWidgetHTML))
	}
}

func TestChannelWidgetHTMLUnder32KiB(t *testing.T) {
	if len(channelWidgetHTML) > 32768 {
		t.Fatalf("channelWidgetHTML %d > 32768", len(channelWidgetHTML))
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
	if sc != "" || act != "refresh" {
		t.Fatalf("legacy got %q %q", sc, act)
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
