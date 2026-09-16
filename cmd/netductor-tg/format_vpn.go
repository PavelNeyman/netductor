package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	qrcode "github.com/skip2/go-qrcode"
)

func formatVPNListPretty(raw string) string {
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>#</th><th>user</th><th>state</th></tr>" + nl)
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 1 {
			parts = strings.Fields(line)
		}
		name := parts[0]
		if name == "relay-uplink" {
			continue
		}
		en := ""
		if len(parts) > 1 {
			en = parts[1]
		}
		n++
		icon := "🟢"
		if en == "off" {
			icon = "🔴"
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s %s</td></tr>"+nl, n, esc(name), icon, esc(en)))
	}
	if n == 0 {
		b.WriteString("<tr><td>—</td><td>—</td><td>—</td></tr>" + nl)
	}
	b.WriteString("</table>")
	return b.String()
}

func formatVPNLinkHTML(name string) (caption string, vless, hy2, sub string) {
	vless = shareURIFrom(runVPN("link", name, "vless"))
	hy2 = shareURIFrom(runVPN("link", name, "hy2"))
	sub = shareURIFrom(runVPN("link", name))
	if sub != "" {
		first := strings.TrimSpace(strings.Split(sub, "\n")[0])
		if strings.HasPrefix(first, "vless://") {
			vless = first
		}
	}
	if strings.Contains(vless, "not found") || strings.Contains(vless, "exit status") {
		vless = ""
	}
	if strings.Contains(hy2, "not found") || strings.Contains(hy2, "exit status") {
		hy2 = ""
	}
	if strings.Contains(sub, "not found") || strings.Contains(sub, "exit status") {
		sub = ""
	}
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("🔗 <b>VPN · " + esc(name) + "</b>" + nl + nl)
	if vless != "" {
		b.WriteString("<b>VLESS · primary</b>" + nl + "<code>" + esc(vless) + "</code>" + nl + nl)
		coreL := strings.TrimSpace(runVPN("link", name, "core"))
		if coreL != "" && coreL != vless && !strings.Contains(coreL, "not found") {
			b.WriteString("<b>VLESS · core (домашний, быстрее)</b>" + nl + "<code>" + esc(coreL) + "</code>" + nl + nl)
		}
	}
	if hy2 != "" {
		b.WriteString("<b>HY2</b>" + nl + "<code>" + esc(hy2) + "</code>" + nl)
	}
	if vless == "" && hy2 == "" {
		b.WriteString("❌ no links")
	}
	return b.String(), vless, hy2, sub
}

func ensureQRFile(path, payload string) string {
	if payload == "" {
		return ""
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	// Always regenerate — cached PNG may still encode core IP while link is relay.
	if err := qrcode.WriteFile(payload, qrcode.Medium, 512, path); err != nil {
		return ""
	}
	_ = os.Chmod(path, 0o600)
	return path
}


func formatUsersListHTML() string {
	nl := string([]byte{10})
	raw := runVPN("list")
	var b strings.Builder
	b.WriteString("<h3>👥 " + esc(T("users")) + "</h3>" + nl)
	b.WriteString("<p><i>" + T("users_hint") + "</i></p>" + nl)
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 1 {
			parts = strings.Fields(line)
		}
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		name := parts[0]
		if name == "relay-uplink" {
			continue
		}
		en := ""
		if len(parts) > 1 {
			en = parts[1]
		}
		n++
		icon := "🟢"
		if en == "off" {
			icon = "🔴"
		}
		// One visual row: status + name, then in-body rich buttons (Bot API 10.3)
		b.WriteString("<p>" + icon + " <b>" + esc(name) + "</b>")
		if en != "" {
			b.WriteString(" · <code>" + esc(en) + "</code>")
		}
		b.WriteString("</p>" + nl)
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(`<tg-button type="callback_data" style="primary" data="u:open:` + name + `">` + esc(T("user_card")) + `</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" data="u:access:` + name + `:vless">` + esc(T("user_access")) + `</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" data="u:rename:` + name + `">` + esc(T("vpn_rename")) + `</tg-button>`)
		b.WriteString(`</tg-button-row>` + nl)
		if n >= 25 {
			break
		}
	}
	if n == 0 {
		b.WriteString("<p>—</p>" + nl)
	}
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="success" data="m:vpn_add">` + esc(T("vpn_add")) + `</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	return b.String()
}

func formatUserHubHTML(name string) string {
	nl := string([]byte{10})
	raw := runVPN("list")
	en := "?"
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) == 0 {
			parts = strings.Fields(line)
		}
		if len(parts) > 0 && parts[0] == name {
			if len(parts) > 1 {
				en = parts[1]
			}
			break
		}
	}
	icon := "🟢"
	if en == "off" {
		icon = "🔴"
	}
	var b strings.Builder
	b.WriteString("<h3>👤 " + esc(name) + " " + icon + "</h3>" + nl)
	b.WriteString("<p><code>" + esc(en) + "</code></p>" + nl)
	b.WriteString("<p><i>" + T("user_hub_hint") + "</i></p>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="primary" data="u:access:` + name + `:vless">` + esc(T("user_access")) + `</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="u:rename:` + name + `">` + esc(T("vpn_rename")) + `</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="success" data="u:enable:` + name + `">` + esc(T("vpn_enable")) + `</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="danger" data="u:disable:` + name + `">` + esc(T("vpn_disable")) + `</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="danger" data="u:revoke:` + name + `">` + esc(T("vpn_revoke")) + `</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	return b.String()
}


func accessPayload(name, mode string) (payload, caption string) {
	nl := "\n"
	switch mode {
	case "core":
		payload = shareURIFrom(runVPN("link", name, "core"))
		if payload == "" {
			payload = shareURIFrom(runVPN("link", name, "vless"))
		}
		caption = "🔗 <b>VLESS · core</b> · " + esc(name)
	case "hy2":
		payload = shareURIFrom(runVPN("link", name, "hy2"))
		caption = "📱 <b>HY2</b> · " + esc(name)
	default:
		mode = "vless"
		payload = shareURIFrom(runVPN("link", name, "vless"))
		caption = "🔗 <b>VLESS · secondary</b> · " + esc(name)
	}
	if strings.Contains(payload, "not found") || strings.Contains(payload, "exit status") {
		payload = ""
	}
	if payload == "" {
		caption += nl + "❌ " + T("no_links")
	}
	return payload, caption
}

func showUserAccess(token string, chat int64, msgID int, name, mode string) {
	if mode == "" {
		mode = "vless"
	}
	payload, cap := accessPayload(name, mode)
	kb := userAccessKeyboard(name, mode)
	dir := filepath.Join("/etc/netductor/clients", name)
	_ = os.MkdirAll(dir, 0o700)

	first := ""
	if payload != "" {
		first = strings.TrimSpace(strings.Split(payload, "\n")[0])
	}

	// Always replace previous message — editMessageMedia is flaky text↔photo.
	if msgID > 0 {
		_ = deleteMessage(token, chat, msgID)
	}

	path := ""
	if first != "" {
		switch mode {
		case "hy2":
			path = ensureQRFile(filepath.Join(dir, "qr-hy2.png"), first)
		case "core":
			path = ensureQRFile(filepath.Join(dir, "qr-core.png"), first)
		default:
			path = ensureQRFile(filepath.Join(dir, "qr-vless.png"), first)
		}
	}

	if path != "" {
		if err := sendPhotoFile(token, chat, path, cap, kb); err != nil {
			fmt.Fprintln(os.Stderr, "sendPhotoFile:", err)
			sendHTML(token, chat, cap, kb)
		}
	} else {
		sendHTML(token, chat, cap, kb)
	}

	// URI as monospace block only (copy-friendly). No custom URL schemes —
	// Telegram often blocks them; QR + long-press copy is reliable.
	if first != "" {
		sendHTML(token, chat, "<code>"+esc(first)+"</code>", kb)
	}
}

func deliverVPNLink(token string, chat int64, msgID int, name string) {
	// Replace the message that contained the user button (list / card).
	showVPNQR(token, chat, msgID, name, "vless", msgID > 0)
}

func vpnQRCaption(name, mode, vless, hy2 string) string {
	ru := getLang() != "en"
	nl := string([]byte{10})
	var b strings.Builder
	if mode == "hy2" {
		b.WriteString("📱 <b>Hysteria2</b> · " + esc(name) + nl + nl)
		if hy2 != "" {
			b.WriteString("<code>" + esc(hy2) + "</code>")
		}
	} else {
		b.WriteString("📱 <b>VLESS Reality</b> · " + esc(name) + nl + nl)
		if vless != "" {
			b.WriteString("<code>" + esc(vless) + "</code>")
		}
		if hy2 != "" {
			if ru {
				b.WriteString(nl + nl + "<i>HY2 — кнопка «HY2 QR» выше</i>")
			} else {
				b.WriteString(nl + nl + "<i>HY2 — use «HY2 QR» button</i>")
			}
		}
	}
	return b.String()
}

func showVPNQR(token string, chat int64, msgID int, name, mode string, edit bool) {
	_, vless, hy2, sub := formatVPNLinkHTML(name)
	if mode != "hy2" {
		mode = "vless"
	}
	kb := userCardKeyboardMode(name, mode)
	dir := filepath.Join("/etc/netductor/clients", name)
	var path, payload string
	if mode == "hy2" {
		payload = hy2
		path = ensureQRFile(filepath.Join(dir, "qr-hy2.png"), hy2)
	} else {
		payload = vless
		if payload == "" {
			payload = sub
		}
		path = ensureQRFile(filepath.Join(dir, "qr-vless.png"), vless)
		if path == "" {
			path = ensureQRFile(filepath.Join(dir, "qr.png"), vless)
		}
	}
	cap := vpnQRCaption(name, mode, vless, hy2)
	if payload == "" && path == "" {
		sendHTML(token, chat, cap, kb)
		return
	}
	if path == "" {
		sendHTML(token, chat, cap, kb)
		return
	}
	if edit && msgID > 0 {
		if err := editPhotoFile(token, chat, msgID, path, cap, kb); err != nil {
			// Text list message cannot become a photo — delete and put QR in its place.
			_ = deleteMessage(token, chat, msgID)
			if err2 := sendPhotoFile(token, chat, path, cap, kb); err2 != nil {
				sendHTML(token, chat, cap+string([]byte{10})+"⚠️ <code>"+esc(err2.Error())+"</code>", kb)
			}
		}
		return
	}
	if err := sendPhotoFile(token, chat, path, cap, kb); err != nil {
		sendHTML(token, chat, cap+string([]byte{10})+"⚠️ <code>"+esc(err.Error())+"</code>", kb)
	}
}

