package main

import (
	"github.com/PavelNeyman/netductor/internal/vpn"
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
	b.WriteString("<b>👥 " + esc(T("users")) + "</b>" + nl)
	b.WriteString("<i>" + T("users_hint") + "</i>" + nl)
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
		b.WriteString(fmt.Sprintf("%s <b>%s</b>", icon, esc(name)))
		if en != "" {
			b.WriteString(" · <code>" + esc(en) + "</code>")
		}
		b.WriteString(nl)
	}
	if n == 0 {
		b.WriteString("<i>—</i>" + nl)
	}
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
	b.WriteString("<b>👤 " + esc(name) + " " + icon + "</b>" + nl)
	b.WriteString("<code>" + esc(en) + "</code>" + nl)
	b.WriteString("<i>" + T("user_hub_hint") + "</i>" + nl)
	lim := vpn.SoftLimitGB(name)
	b.WriteString(fmt.Sprintf("Soft limit: <code>%.0f</code> GiB (0=off)"+nl, lim))
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
	// Prefer /etc/netductor/netductor.conf (REDIRECT_BASE) via ndconfig.Load, or env.
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return ""
}

func importRedirectURL(deep string) string {
	deep = strings.TrimSpace(deep)
	base := redirectBase()
	if deep == "" || base == "" {
		return ""
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return ""
	}
	enc := base64.RawURLEncoding.EncodeToString([]byte(deep))
	return strings.TrimRight(base, "/") + "/r?u=" + enc
}


// showWorkProfileButton: SR Config Mac+OC profile is for the fleet operator account.
// Install used name "Pavel", not "operator".
func showWorkProfileButton(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "operator" || n == "pavel" {
		return true
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("NETDUCTOR_WORK_PROFILE_USER"))); v != "" && n == v {
		return true
	}
	return false
}


func formatAccessRichHTML(name, mode, uri string) string {
	nl := string([]byte{10})
	title := "VLESS · secondary"
	switch mode {
	case "core":
		title = "VLESS · core"
	case "hy2":
		title = "HY2"
	}
	var b strings.Builder
	b.WriteString("<b>🔗 " + esc(title) + " · " + esc(name) + "</b>" + nl)
	if uri != "" {
		b.WriteString("<pre><code>" + esc(uri) + "</code></pre>")
	} else {
		b.WriteString("❌ " + T("no_links"))
	}
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




func sendWorkProfileDocument(token string, chat int64) {
	candidates := []string{
		"/opt/netductor/profiles/nd-oc.conf",
		"/etc/netductor/profiles/nd-oc.conf",
	}
	var path string
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			path = c
			break
		}
	}
	if path == "" {
		sendHTML(token, chat, "❌ SR Config profile file not found on server", nil)
		return
	}
	nl := string([]byte{10})
	cap := "📥 <b>SR Config</b>" + nl + "Shadowrocket → Config → import this file." + nl + "OpenConnect first, then Config mode."
	if err := sendDocumentFile(token, chat, path, cap); err != nil {
		fmt.Fprintln(os.Stderr, "sendWorkProfileDocument:", err)
		sendHTML(token, chat, "❌ Failed to send profile: "+esc(err.Error()), nil)
	}
}




