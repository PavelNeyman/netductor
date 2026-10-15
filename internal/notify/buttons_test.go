package notify

import (
	"strings"
	"testing"
)

func TestExtractTGButtons(t *testing.T) {
	msg := "🆕 <b>Netductor update</b>\nlocal <code>0.9.1</code> → latest <code>0.9.2</code>\n" +
		`<tg-button-row><tg-button type="callback_data" style="primary" data="m:updates">🔄 Updates</tg-button></tg-button-row>`
	text, kb := extractTGButtons(msg)
	if strings.Contains(text, "tg-button") {
		t.Fatalf("tags left in text: %q", text)
	}
	if !strings.Contains(text, "Netductor update") {
		t.Fatalf("body lost: %q", text)
	}
	if kb == "" || !strings.Contains(kb, "m:updates") || !strings.Contains(kb, "Updates") {
		t.Fatalf("markup: %q", kb)
	}
}

func TestExtractTGButtonsNone(t *testing.T) {
	text, kb := extractTGButtons("plain <b>html</b>")
	if kb != "" {
		t.Fatal(kb)
	}
	if !strings.Contains(text, "plain") {
		t.Fatal(text)
	}
}
