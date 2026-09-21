package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/mtls"
)

func handleCertsCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:certs" && !strings.HasPrefix(data, "m:certs:") {
		return false
	}
	ru := getLang() != "en"
	var b strings.Builder
	if ru {
		b.WriteString("🔐 <b>Сертификаты mTLS</b>\n")
	} else {
		b.WriteString("🔐 <b>mTLS certificates</b>\n")
	}
	b.WriteString("\n<b>Plane</b>\n")
	for _, pc := range mtls.ListPlaneCerts() {
		tag := "🟢"
		if pc.DaysLeft < 0 {
			tag = "🔴"
		} else if pc.DaysLeft <= 30 {
			tag = "🟡"
		}
		fmt.Fprintf(&b, "%s <code>%s</code> %dd · %s\n", tag, esc(pc.Name), pc.DaysLeft, pc.NotAfter.UTC().Format("2006-01-02"))
	}
	b.WriteString("\n<b>Clients</b>\n")
	clients, err := mtls.ListClientCerts()
	if err != nil || len(clients) == 0 {
		if ru {
			b.WriteString("<i>нет client certs</i>\n")
		} else {
			b.WriteString("<i>no client certs</i>\n")
		}
	} else {
		for _, c := range clients {
			tag := "🟢"
			if c.Revoked {
				tag = "⛔"
			} else if c.DaysLeft < 0 {
				tag = "🔴"
			} else if c.DaysLeft <= 30 {
				tag = "🟡"
			}
			fmt.Fprintf(&b, "%s <code>%s</code> %dd · %s\n", tag, esc(c.NodeID), c.DaysLeft, c.NotAfter.UTC().Format("2006-01-02"))
		}
	}
	if pend, err := mtls.PendingRotates(); err == nil && len(pend) > 0 {
		b.WriteString("\n<b>Pending rotate</b>\n")
		for _, p := range pend {
			fmt.Fprintf(&b, "• %s old=%s new=%s\n", esc(p.NodeID), esc(p.OldSerial), esc(p.NewSerial))
		}
	}
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("🔄", "m:certs", ""), btn(T("main_menu"), "m:menu", "primary")},
	}}
	reply(token, chat, msgID, b.String(), kb)
	return true
}
