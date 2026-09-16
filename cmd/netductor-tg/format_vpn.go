package main

import (
	"encoding/base64"
	"net/url"
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


func accessPayload(name, mode string) (payload string) {
	switch mode {
	case "core":
		payload = shareURIFrom(runVPN("link", name, "core"))
		if payload == "" {
			payload = shareURIFrom(runVPN("link", name, "vless"))
		}
	case "hy2":
		payload = shareURIFrom(runVPN("link", name, "hy2"))
	default:
		mode = "vless"
		payload = shareURIFrom(runVPN("link", name, "vless"))
	}
	if strings.Contains(payload, "not found") || strings.Contains(payload, "exit status") {
		payload = ""
	}
	return strings.TrimSpace(strings.Split(payload, "\n")[0])
}

// formatAccessRichHTML — body actions only (TG-UI.md). Navigation via reply_markup.
func redirectBase() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE")); v != "" {
		return strings.TrimRight(v, "/")
	}
	// default: primary public IP http (port 80 redirect-serve)
	return "http://2.27.118.70"
}

func importRedirectURL(deep string) string {
	deep = strings.TrimSpace(deep)
	if deep == "" {
		return ""
	}
	enc := base64.RawURLEncoding.EncodeToString([]byte(deep))
	return redirectBase() + "/r?u=" + enc
}

func formatAccessRichHTML(name, mode, uri string) string {
	nl := "\n"
	title := "VLESS · secondary"
	switch mode {
	case "core":
		title = "VLESS · core"
	case "hy2":
		title = "HY2"
	}
	styleV, styleC, styleH := "", "", ""
	switch mode {
	case "core":
		styleC = ` style="primary"`
	case "hy2":
		styleH = ` style="primary"`
	default:
		styleV = ` style="primary"`
	}
	var b strings.Builder
	b.WriteString("<h3>🔗 " + esc(title) + " · " + esc(name) + "</h3>" + nl)
	if uri != "" {
		b.WriteString(`<img src="tg://photo?id=qr1"/>` + nl)
		b.WriteString("<pre><code>" + esc(uri) + "</code></pre>" + nl)
		enc := url.PathEscape(uri)
		sr := importRedirectURL("shadowrocket://add/" + enc)
		happ := importRedirectURL("happ://add/" + enc)
		incy := importRedirectURL("incy://add/" + enc)
		// attr escape
		attr := func(s string) string {
			return strings.ReplaceAll(s, "&", "&amp;")
		}
		b.WriteString(`<tg-button-row align="left">`)
		if sr != "" {
			b.WriteString(`<tg-button type="url" url="` + attr(sr) + `">Shadowrocket</tg-button>`)
		}
		if happ != "" {
			b.WriteString(`<tg-button type="url" url="` + attr(happ) + `">Happ</tg-button>`)
		}
		if incy != "" {
			b.WriteString(`<tg-button type="url" url="` + attr(incy) + `">INCY</tg-button>`)
		}
		b.WriteString(`</tg-button-row>` + nl)
	} else {
		b.WriteString("<p>❌ " + T("no_links") + "</p>" + nl)
	}
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data"` + styleV + ` data="u:access:` + name + `:vless">VLESS</tg-button>`)
	b.WriteString(`<tg-button type="callback_data"` + styleC + ` data="u:access:` + name + `:core">Core</tg-button>`)
	b.WriteString(`<tg-button type="callback_data"` + styleH + ` data="u:access:` + name + `:hy2">HY2</tg-button>`)
	b.WriteString(`</tg-button-row>`)
	return b.String()
}


func sendAppDeepLink(token string, chat int64, name, mode, client string) {
	uri := accessPayload(name, mode)
	if uri == "" {
		sendHTML(token, chat, "❌ "+T("no_links"), nil)
		return
	}
	enc := url.PathEscape(uri)
	var deep, label string
	switch client {
	case "happ":
		deep, label = "happ://add/"+enc, "Happ"
	case "incy":
		deep, label = "incy://add/"+enc, "INCY"
	default:
		deep, label = "shadowrocket://add/"+enc, "Shadowrocket"
	}
	// Prefer HTTP redirect button URL if redirect-serve is up
	if link := importRedirectURL(deep); link != "" {
		msg := "📲 <b>" + esc(label) + "</b>\n"
		msg += `<tg-button-row><tg-button type="url" url="` + strings.ReplaceAll(link, "&", "&amp;") + `">` + esc(label) + `</tg-button></tg-button-row>`
		sendHTML(token, chat, msg, nil)
		return
	}
	msg := "📲 <b>" + esc(label) + "</b>\n<pre><code>" + esc(deep) + "</code></pre>\n"
	msg += "<i>Скопируйте и откройте на телефоне.</i>"
	sendHTML(token, chat, msg, nil)
}

func showUserAccess(token string, chat int64, msgID int, name, mode string) {
	if mode == "" {
		mode = "vless"
	}
	uri := accessPayload(name, mode)
	html := formatAccessRichHTML(name, mode, uri)
	kb := userAccessKeyboard(name, mode) // navigation only under message
	dir := filepath.Join("/etc/netductor/clients", name)
	_ = os.MkdirAll(dir, 0o700)

	if uri == "" {
		reply(token, chat, msgID, html, kb)
		return
	}
	qrPath := filepath.Join(dir, "qr-vless.png")
	switch mode {
	case "hy2":
		qrPath = filepath.Join(dir, "qr-hy2.png")
	case "core":
		qrPath = filepath.Join(dir, "qr-core.png")
	}
	if p := ensureQRFile(qrPath, uri); p == "" {
		fmt.Fprintln(os.Stderr, "ensureQRFile failed for", name, mode)
		reply(token, chat, msgID, html, kb)
		return
	}
	replyRichWithPhoto(token, chat, msgID, html, qrPath, "qr1", kb)
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

