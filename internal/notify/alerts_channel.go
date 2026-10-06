package notify

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

const alertsChatSecret = "telegram_alerts_chat_id"

// AlertsChatID returns dedicated alerts channel id (empty = admin chat + optional topics).
func AlertsChatID() string {
	if v := strings.TrimSpace(secret(alertsChatSecret)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("NETDUCTOR_TG_ALERTS_CHAT"))
}

// AlertsChannelConfigured is true when alerts go to a separate chat (no topic routing).
func AlertsChannelConfigured() bool {
	return AlertsChatID() != ""
}

// SetAlertsChatID stores channel id (e.g. -100…). Empty clears.
func SetAlertsChatID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ClearAlertsChatID()
	}
	// basic validation: digits, optional leading -
	s := id
	if strings.HasPrefix(s, "-") {
		s = s[1:]
	}
	if s == "" {
		return fmt.Errorf("invalid chat id")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return fmt.Errorf("chat id must be numeric (got %q)", id)
		}
	}
	return paths.WriteSecret(alertsChatSecret, id)
}

// ClearAlertsChatID removes the secret file.
func ClearAlertsChatID() error {
	p := paths.EtcDir() + "/secrets/" + alertsChatSecret
	_ = os.Remove(p)
	return nil
}

// AlertsRoutingStatus is JSON/UI friendly.
type AlertsRoutingStatus struct {
	Mode         string `json:"mode"` // channel | topics | admin_only
	AlertsChatID string `json:"alerts_chat_id,omitempty"`
	AdminChatID  string `json:"admin_chat_id,omitempty"`
	ChannelSet   bool   `json:"channel_set"`
	Hint         string `json:"hint,omitempty"`
}

// GetAlertsRoutingStatus describes where alerts go.
func GetAlertsRoutingStatus() AlertsRoutingStatus {
	st := AlertsRoutingStatus{
		AdminChatID:  secret("telegram_admin_id"),
		AlertsChatID: AlertsChatID(),
	}
	st.ChannelSet = st.AlertsChatID != ""
	if st.ChannelSet {
		st.Mode = "channel"
		st.Hint = "Alerts → channel; menu/VPN cards stay in operator DM"
	} else if ThreadID("alerts") > 0 {
		st.Mode = "topics"
		st.Hint = "Alerts → forum topics in admin chat (set telegram_alerts_chat_id for a channel)"
	} else {
		st.Mode = "admin_only"
		st.Hint = "Alerts → admin chat General; set telegram_alerts_chat_id for a dedicated channel"
	}
	return st
}

// AlertsRoutingHTML for TG UI.
func AlertsRoutingHTML(ru bool) string {
	st := GetAlertsRoutingStatus()
	if ru {
		var b strings.Builder
		b.WriteString("📢 <b>Куда уходят алерты</b>\n")
		b.WriteString("Режим: <code>" + st.Mode + "</code>\n")
		if st.AlertsChatID != "" {
			b.WriteString("Канал: <code>" + st.AlertsChatID + "</code>\n")
		} else {
			b.WriteString("Канал: <i>не задан</i>\n")
		}
		if st.AdminChatID != "" {
			b.WriteString("Admin DM: <code>" + st.AdminChatID + "</code>\n")
		}
		b.WriteString("\n<code>/alerts_chat -100…</code> — задать канал\n")
		b.WriteString("<code>/alerts_chat clear</code> — сбросить\n")
		b.WriteString("<code>/alerts_chat test</code> — пробное сообщение\n")
		return b.String()
	}
	var b strings.Builder
	b.WriteString("📢 <b>Alert routing</b>\n")
	b.WriteString("Mode: <code>" + st.Mode + "</code>\n")
	if st.AlertsChatID != "" {
		b.WriteString("Channel: <code>" + st.AlertsChatID + "</code>\n")
	} else {
		b.WriteString("Channel: <i>not set</i>\n")
	}
	if st.AdminChatID != "" {
		b.WriteString("Admin DM: <code>" + st.AdminChatID + "</code>\n")
	}
	b.WriteString("\n<code>/alerts_chat -100…</code> — set channel\n")
	b.WriteString("<code>/alerts_chat clear</code> — clear\n")
	b.WriteString("<code>/alerts_chat test</code> — send test\n")
	return b.String()
}
