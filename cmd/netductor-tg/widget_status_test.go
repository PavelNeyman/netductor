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
